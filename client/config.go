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

package client

// Config is used for spawning a client, it contains a set HTTP API server's
// configuration. TCPAddr will be used if both specified.
type Config struct {
	UnixAddr string
	TCPAddr  string
}

// Apply generates a new Client instance, and returns encountered error if any.
func (in *Config) Apply() (*Client, error) {
	if in.TCPAddr != "" {
		return NewClient("tcp", in.TCPAddr)
	}
	return NewClient("unix", in.UnixAddr)
}

// NewConfig creates a new Config
func NewConfig() *Config {
	return new(Config)
}
