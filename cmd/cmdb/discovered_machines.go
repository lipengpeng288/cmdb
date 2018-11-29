// Copyright 2018 Alfred Chou <unioverlord@gmail.com>
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

package main

import (
	"fmt"
	"os"

	cobra "github.com/spf13/cobra"
)

// UpdateDiscoveredMachinesCmd represents the discovered command of cmdb
var UpdateDiscoveredMachinesCmd = &cobra.Command{
	Use:   "discovered",
	Short: "Trigger a discovered machines update immediately",
	Long:  `Trigger a discovered machines update immediately.`,
	Run: func(cmd *cobra.Command, args []string) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "%v\n", r)
				os.Exit(2)
			}
		}()
		cli, err := config.Apply()
		if err != nil {
			panic(err)
		}
		err = cli.UpdateDiscoveredMachines()
		if err != nil {
			panic(err)
		}
	},
}

// GetDiscoveredMachinesCmd represents the discovered command of cmdb
var GetDiscoveredMachinesCmd = &cobra.Command{
	Use:   "discovered",
	Short: "Get the discovered machines from the backend.",
	Long:  `Get the discovered machines from the backend.`,
	Run: func(cmd *cobra.Command, args []string) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "%v\n", r)
				os.Exit(2)
			}
		}()
		cli, err := config.Apply()
		if err != nil {
			panic(err)
		}
		err = cli.GetDiscovertedMachines()
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	UpdateCmd.AddCommand(UpdateDiscoveredMachinesCmd)
	GetCmd.AddCommand(GetDiscoveredMachinesCmd)
}
