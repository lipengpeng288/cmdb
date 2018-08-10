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

package cli

import (
	"bufio"
	"fmt"
	"io"
)

// ComfirmPrompt is a prompt to enforce user to make a explicit decision.
type ComfirmPrompt struct {
	writer io.Writer
	reader io.Reader
}

// Enforce print hints and await user's choice, will never return unless a valid choice was detected.
func (in *ComfirmPrompt) Enforce() (yes bool) {
	fmt.Fprintf(in.writer, "Confirm? (y/n) ")
	scanner := bufio.NewScanner(in.reader)
LOOP:
	for scanner.Scan() {
		input := scanner.Text()
		switch input {
		case "Y", "y":
			yes = true
			break LOOP
		case "N", "n":
			break LOOP
		}
		fmt.Fprintf(in.writer, "Confirm? (y/n) ")
	}
	return
}

// NewComfirmPrompt returns a new ComfirmPrompt
func NewComfirmPrompt(out io.Writer, in io.Reader) *ComfirmPrompt {
	return &ComfirmPrompt{
		writer: out,
		reader: in,
	}
}
