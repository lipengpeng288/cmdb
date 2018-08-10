// Copyright © 2018 Alfred Chou <unioverlord@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	toml "github.com/pelletier/go-toml"
	executor "github.com/universonic/cmdb/server/executor"
	scheduler "github.com/universonic/cmdb/server/scheduler"
	web "github.com/universonic/cmdb/server/web"
	storage "github.com/universonic/cmdb/shared/storage"
	fsutil "github.com/universonic/cmdb/utils/filesystem"
	logging "github.com/universonic/cmdb/utils/logging"
	zapcore "go.uber.org/zap/zapcore"
	yaml "gopkg.in/yaml.v2"
)

// ConfigInitiator is a parser that is used during intialization.
type ConfigInitiator struct {
	Database *storage.ConfigInitiator `json:"database,omitempty" yaml:"database,omitempty" toml:"database,omitempty"`
}

// Ready returns a completed configuration parser and any encountered error.
func (in *ConfigInitiator) Ready() (*Config, error) {
	dbconf, err := storage.NewQualifiedConfig(in.Database.Adapter)
	if err != nil {
		return nil, err
	}
	cfg := &Config{
		Database: dbconf,
	}
	return cfg, nil
}

// NewConfigInitiator returns a new empty NewConfigInitiator
func NewConfigInitiator() *ConfigInitiator {
	return new(ConfigInitiator)
}

// Config is the carrier to be used for parsing config file into it.
type Config struct {
	Web       *web.Config              `json:"web,omitempty" yaml:"web,omitempty" toml:"web,omitempty"`
	Scheduler *scheduler.Config        `json:"scheduler,omitempty" yaml:"scheduler,omitempty" toml:"scheduler,omitempty"`
	Executor  *executor.Config         `json:"executor,omitempty" yaml:"executor,omitempty" toml:"executor,omitempty"`
	Database  *storage.QualifiedConfig `json:"database,omitempty" yaml:"database,omitempty" toml:"database,omitempty"`
	Log       *LogConfig               `json:"log,omitempty" yaml:"log,omitempty" toml:"log,omitempty"`
}

// Complete fulfilled empty fields with default values.
func (in *Config) Complete() {
	in.Web.Complete()
	in.Scheduler.Complete()
	in.Executor.Complete()
	in.Log.Complete()
}

// Apply applies the configuration and spawn a new server instance, and returns any encountered error.
func (in *Config) Apply() (svr *Server, err error) {
	dir, lvl := in.Log.Apply()
	var exists bool
	exists, err = fsutil.FileExists(dir)
	if !exists {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return
		}
	}
	if err != nil {
		return
	}
	serverLoggerFact, err := logging.NewFactoryConfig(
		[]string{filepath.Join(dir, "cmdb.log")},
		[]string{filepath.Join(dir, "cmdb-error.log")},
	).Apply(lvl, false)
	if err != nil {
		return nil, err
	}
	serverLogger := serverLoggerFact.Merge(logging.DefaultLoggerFactory).New().Sugar().Named("CMDB")
	storageLoggerFact, err := logging.NewFactoryConfig(
		[]string{filepath.Join(dir, "storage.log")},
		[]string{filepath.Join(dir, "storage-error.log")},
	).Apply(lvl, false)
	if err != nil {
		return nil, err
	}
	defer serverLogger.Sync()
	storageLogger := storageLoggerFact.Merge(logging.DefaultLoggerFactory).New().Sugar().Named("STORAGE")
	storage, err := in.Database.Open(storageLogger)
	if err != nil {
		return nil, err
	}
	webServerLoggerFact, err := logging.NewFactoryConfig(
		[]string{filepath.Join(dir, "web.log")},
		[]string{filepath.Join(dir, "web-error.log")},
	).Apply(lvl, false)
	if err != nil {
		return nil, err
	}
	webServerLogger := webServerLoggerFact.Merge(logging.DefaultLoggerFactory).New().Sugar().Named("WEB")
	webServer, err := in.Web.Apply()
	if err != nil {
		return nil, err
	}
	webServer.Prepare(storage, webServerLogger)
	schedulerLoggerFact, err := logging.NewFactoryConfig(
		[]string{filepath.Join(dir, "scheduler.log")},
		[]string{filepath.Join(dir, "scheduler-error.log")},
	).Apply(lvl, false)
	if err != nil {
		return nil, err
	}
	schedulerLogger := schedulerLoggerFact.Merge(logging.DefaultLoggerFactory).New().Sugar().Named("SCHEDULER")
	schedulerServer, err := in.Scheduler.Apply()
	if err != nil {
		return nil, err
	}
	schedulerServer.Prepare(storage, schedulerLogger)
	executorLoggerFact, err := logging.NewFactoryConfig(
		[]string{filepath.Join(dir, "executor.log")},
		[]string{filepath.Join(dir, "executor-error.log")},
	).Apply(lvl, false)
	if err != nil {
		return nil, err
	}
	executorLogger := executorLoggerFact.Merge(logging.DefaultLoggerFactory).New().Sugar().Named("EXECUTOR")
	executorServers, err := in.Executor.Apply()
	if err != nil {
		return nil, err
	}
	for i := range executorServers {
		schedulerServer.Handler.RegisterExecutor(executorServers[i])
		executorServers[i].Prepare(storage, executorLogger.Named(fmt.Sprintf("#%d", i+1)))
	}
	dAtA, _ := json.MarshalIndent(in, "", "  ")
	serverLogger.Debugf("Configuration => %s", dAtA)
	return &Server{
		storage:   storage,
		closeCh:   make(chan error, 1),
		logger:    serverLogger,
		web:       webServer,
		scheduler: schedulerServer,
		executors: executorServers,
	}, nil
}

// LogConfig indicates the logging config.
type LogConfig struct {
	Output string `json:"output,omitempty" yaml:"output,omitempty" toml:"output,omitempty"`
	Level  int8   `json:"level,omitempty" yaml:"level,omitempty" toml:"level,omitempty"`
}

// Complete fulfilled empty fields with default values.
func (in *LogConfig) Complete() {
	if in.Output == "" {
		in.Output = "/var/log/cmdb"
	}
}

// Apply applies the configuration and returns a set of logging config.
func (in *LogConfig) Apply() (string, zapcore.Level) {
	switch in.Level {
	case 1:
		return in.Output, logging.DEBUG
	case 2:
		return in.Output, logging.INFO
	case 3:
		return in.Output, logging.WARN
	case 4:
		return in.Output, logging.ERROR
	}
	return in.Output, logging.INFO
}

// ParseFromFile parses config from given file and try to spawn a new server, returns any encountered error.
func ParseFromFile(f string) (*Server, error) {
	fi, err := os.Open(f)
	if err != nil {
		return nil, err
	}
	defer fi.Close()

	var buf bytes.Buffer
	if _, err = io.Copy(&buf, fi); err != nil {
		return nil, err
	}
	dAtA := buf.Bytes()

	var (
		initiator = NewConfigInitiator()
		cfg       *Config
	)
	switch ext := filepath.Ext(fi.Name()); ext {
	case ".toml":
		if err = toml.Unmarshal(dAtA, initiator); err != nil {
			return nil, err
		}
		if cfg, err = initiator.Ready(); err != nil {
			return nil, err
		}
		if err = toml.Unmarshal(dAtA, cfg); err != nil {
			return nil, err
		}
	case ".yaml", ".yml":
		if err = yaml.Unmarshal(dAtA, initiator); err != nil {
			return nil, err
		}
		if cfg, err = initiator.Ready(); err != nil {
			return nil, err
		}
		if err = yaml.Unmarshal(dAtA, cfg); err != nil {
			return nil, err
		}
	case ".json":
		if err = json.Unmarshal(dAtA, initiator); err != nil {
			return nil, err
		}
		if cfg, err = initiator.Ready(); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(dAtA, cfg); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("Not supported extension: %s", ext)
	}
	if cfg.Web == nil {
		cfg.Web = new(web.Config)
	}
	if cfg.Scheduler == nil {
		cfg.Scheduler = new(scheduler.Config)
	}
	if cfg.Executor == nil {
		cfg.Executor = new(executor.Config)
	}
	if cfg.Database == nil {
		cfg.Database = new(storage.QualifiedConfig)
	}
	if cfg.Log == nil {
		cfg.Log = new(LogConfig)
	}
	cfg.Complete()
	return cfg.Apply()
}
