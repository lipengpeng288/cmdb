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

package executor

// Config is the configuration of executor
type Config struct {
	Timeout int
	Workers int
}

// Complete fulfills the empty fields of Config
func (in *Config) Complete() {
	if in.Timeout <= 0 {
		in.Timeout = 3600
	}
	if in.Workers <= 0 {
		in.Workers = 1
	}
}

// Apply spawns a new API server with configuration, and returns any encountered error.
func (in *Config) Apply() ([]*Executor, error) {
	var executors []*Executor
	for i := 0; i < in.Workers; i++ {
		executors = append(executors, NewExecutor(in.Timeout))
	}
	return executors, nil
}
