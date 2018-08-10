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

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// ansibleError is a qualified ansible error carrier
type ansibleError struct {
	ss map[string]string
}

func (e ansibleError) Error() string {
	if len(e.ss) == 1 {
		for k, v := range e.ss {
			return fmt.Sprintf("Execution failure on host %s: %s", k, v)
		}
	}
	dAtA, _ := json.Marshal(e.ss)
	return fmt.Sprintf("Multiple errors occured: %s", dAtA)
}

// newAnsibleError initiate a new ansible error instance.
func newAnsibleError() *ansibleError {
	return &ansibleError{
		ss: make(map[string]string),
	}
}

// Ansible is a port of ansible to run task for generic purpose.
type Ansible struct {
	fast          bool
	Module        string
	InventoryFile string
	Stdout        []byte
	Stderr        []byte
	Result        map[string][]byte
}

// Execute starts worker and returns any encountered error.
func (in *Ansible) Execute() error {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	cmd := exec.Command("ansible", "all", "-m", in.Module, "-i", in.InventoryFile, "-t", dir, "-vvv")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	defer stdout.Close()
	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}
	defer stderr.Close()
	err = cmd.Start()
	if err != nil {
		panic(err)
	}
	go func() {
		in.Stdout, _ = ioutil.ReadAll(stdout)
	}()
	go func() {
		in.Stderr, _ = ioutil.ReadAll(stderr)
	}()
	err = cmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				switch status.ExitStatus() {
				case 2, 4:
					// Common failure
					if in.fast {
						return fmt.Errorf("Failure on ansible module '%s': %v", in.Module, err)
					}
					goto FINALIZE
				case 127:
					return fmt.Errorf("Ansible could not be found in your PATH")
				}
			}
		}
		return fmt.Errorf("Unexpected error: %v", err)
	}
FINALIZE:
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	multiError := newAnsibleError()
	for _, each := range files {
		if !each.IsDir() {
			fi, err := os.Open(filepath.Join(dir, each.Name()))
			if err != nil {
				multiError.ss[each.Name()] = err.Error()
				continue
			}
			buf := new(bytes.Buffer)
			_, err = io.Copy(buf, fi)
			fi.Close()
			if err != nil {
				multiError.ss[each.Name()] = err.Error()
				continue
			}

			in.Result[each.Name()] = buf.Bytes()
			cv := new(AnsibleValidator)
			err = json.Unmarshal(buf.Bytes(), cv)
			if err != nil {
				multiError.ss[each.Name()] = "Result was issued but is not in valid JSON format. Has ansible returned gracefully?"
				continue
			}
			if cv.Message != "" {
				multiError.ss[each.Name()] = cv.Message
			}
		}
	}
	if len(multiError.ss) != 0 {
		return fmt.Errorf("Something went wrong with ansible module '%s': %v", in.Module, multiError)
	}
	return nil
}

// NewAnsible initiates a new ansible executor with given module name and inventory file. If fast mode
// is enabled (disabled by default), will return immediately rather than collect result data and then
// return with a qualified error.
func NewAnsible(moduleName, inventoryFile string, fast ...bool) *Ansible {
	var fastMode bool
	if len(fast) != 0 && fast[0] {
		fastMode = true
	}
	return &Ansible{
		fast:          fastMode,
		Module:        moduleName,
		InventoryFile: inventoryFile,
		Result:        make(map[string][]byte),
	}
}

// AnsibleValidator is used for validating the execution state on a host.
type AnsibleValidator struct {
	Changed bool   `json:"changed,omitempty"`
	Message string `json:"msg,omitempty"`
}

// GetFailedNodesFromError try to return a list of node within the given error if it represents
// a ansibleError.
func GetFailedNodesFromError(e error) (nodes []string) {
	qe, ok := e.(*ansibleError)
	if ok {
		for i := range qe.ss {
			nodes = append(nodes, i)
		}
	}
	return
}
