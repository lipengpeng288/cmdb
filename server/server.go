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
	"os"
	"os/signal"
	"syscall"

	executor "github.com/universonic/cmdb/server/executor"
	scheduler "github.com/universonic/cmdb/server/scheduler"
	web "github.com/universonic/cmdb/server/web"
	genericStorage "github.com/universonic/cmdb/shared/storage/generic"
	zap "go.uber.org/zap"
)

// Server indicates the main server process
type Server struct {
	storage   genericStorage.Storage
	closeCh   chan error
	logger    *zap.SugaredLogger
	web       *web.Server
	scheduler *scheduler.Server
	executors []*executor.Executor
}

// Run start the server as a main process.
func (in *Server) Run() error {
	defer in.logger.Sync()
	defer in.logger.Info("Server exited")

	var executorCltz []<-chan struct{}
	for i := range in.executors {
		executorCltz = append(executorCltz, in.executors[i].Subscribe())
	}
	schedulerClz := in.scheduler.Subscribe()
	apiClz := in.web.Subscribe()

	in.logger.Info("Server is starting...")
	for i := range in.executors {
		go func(exec *executor.Executor) {
			err := exec.Serve()
			if err != nil {
				in.closeCh <- err
			}
		}(in.executors[i])
	}
	go func() {
		err := in.scheduler.Serve()
		if err != nil {
			in.closeCh <- err
		}
	}()
	go func() {
		err := in.web.Serve()
		if err != nil {
			in.closeCh <- err
		}
	}()
	in.logger.Info("Server is ready.")
	in.logger.Sync()

	clz := make(chan os.Signal, 1)
	defer close(clz)
	// We accept graceful shutdowns when quit via SIGINT (Ctrl+C) and SIGTERM (Ctrl+/).
	// SIGKILL or SIGQUIT will not be caught.
	signal.Notify(clz, syscall.SIGINT, syscall.SIGTERM)

LOOP:
	for {
		select {
		case <-clz:
			break LOOP
		case e := <-in.closeCh:
			if e != nil {
				in.logger.Error(e)
			} else {
				in.logger.Fatal("Subprocess exited before server is terminated!")
			}
			break LOOP
		}
	}

	in.web.Stop()
	in.scheduler.Stop()
	for i := range in.executors {
		in.executors[i].Stop()
	}
	defer in.storage.Close()
	<-apiClz
	<-schedulerClz
	for i := range executorCltz {
		<-executorCltz[i]
	}
	return nil
}
