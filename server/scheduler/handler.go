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

package scheduler

import (
	"math/rand"
	"time"

	executor "github.com/universonic/cmdb/server/executor"
	genericStorage "github.com/universonic/cmdb/shared/storage/generic"
	zap "go.uber.org/zap"
)

// Handler is indeed an external executer caller.
type Handler struct {
	executor []*executor.Executor
	storage  genericStorage.Storage
	logger   *zap.SugaredLogger
}

// RegisterExecutor is currently a work around for simplified structure.
func (in *Handler) RegisterExecutor(exec *executor.Executor) {
	in.executor = append(in.executor, exec)
}

func (in *Handler) createMachineDigestOnTime() {
	defer in.logger.Sync()
	digest := genericStorage.NewMachineDigest()
	err := in.storage.Create(digest)
	if err != nil {
		in.logger.Error("Could not create new machine digest due to: %v", err)
	}
}

func (in *Handler) refreshMachineSnapshotOnEvent(event genericStorage.WatchEvent) {
	defer in.logger.Sync()

	in.logger.Info("Notifying executor to refresh machine information on time")
	digest := genericStorage.NewMachineDigest()
	err := event.Unmarshal(digest)
	if err != nil {
		in.logger.Error(err)
		return
	}
	rand.Seed(time.Now().Unix())
	in.executor[rand.Intn(len(in.executor))].NotifyDigest(digest)
}

// NewHandler return a new Handler instance.
func NewHandler(storage genericStorage.Storage, logger *zap.SugaredLogger) *Handler {
	return &Handler{
		storage: storage,
		logger:  logger,
	}
}
