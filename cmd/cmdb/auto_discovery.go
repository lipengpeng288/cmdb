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

// CreateAutoDiscoveryCmd represents the discovery command of cmdb
var CreateAutoDiscoveryCmd = &cobra.Command{
	Use:   "discovery",
	Short: "Create an auto discovery subnet",
	Long:  `Create an auto discovery subnet`,
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
		err = cli.CreateAutoDiscovery(name, cidr)
		if err != nil {
			panic(err)
		}
	},
}

// GetAutoDiscoveryCmd represents the discovery command of cmdb
var GetAutoDiscoveryCmd = &cobra.Command{
	Use:   "discovery",
	Short: "Overview auto discovery configuration",
	Long:  `Overview auto discovery configuration. Possible to query multiple machines at one time.`,
	Run: func(cmd *cobra.Command, args []string) {
		var name []string
		if all {
			name = []string{"*"}
		} else if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "At least one machine name must be given.\n")
			os.Exit(1)
		} else {
			name = args
		}
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
		err = cli.GetAutoDiscovery(name)
		if err != nil {
			panic(err)
		}
	},
}

// UpdateAutoDiscoveryCmd represents the discovery command of cmdb
var UpdateAutoDiscoveryCmd = &cobra.Command{
	Use:   "discovery",
	Short: "Update an auto discovery",
	Long:  `Update an auto discovery`,
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
		err = cli.UpdateAutoDiscovery(name, cidr)
		if err != nil {
			panic(err)
		}
	},
}

// RemoveAutoDiscoveryCmd represents the discovery command of cmdb
var RemoveAutoDiscoveryCmd = &cobra.Command{
	Use:   "discovery",
	Short: "Remove an auto discovery",
	Long:  `Remove an auto discovery`,
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
		err = cli.RemoveAutoDiscovery(name, yes)
		if err != nil {
			panic(err)
		}
	},
}

var (
	cidr string
)

func init() {
	// Create
	CreateCmd.AddCommand(CreateAutoDiscoveryCmd)
	CreateAutoDiscoveryCmd.PersistentFlags().StringVarP(
		&name, "name", "n", name, "The name of the machine to create.",
	)
	CreateMachineCmd.PersistentFlags().StringVarP(
		&cidr, "cidr", "c", cidr, `The CIDR of the subnet to automatically discover machines from.
		Such as: "172.17.1.0/24"`,
	)
	// Get
	GetCmd.AddCommand(GetAutoDiscoveryCmd)
	GetAutoDiscoveryCmd.PersistentFlags().BoolVar(
		&all, "all", all, "Select all existing auto discoveries.",
	)
	// Update
	UpdateCmd.AddCommand(UpdateAutoDiscoveryCmd)
	UpdateAutoDiscoveryCmd.PersistentFlags().StringVarP(
		&name, "name", "n", name, "The name of the machine to update.",
	)
	UpdateAutoDiscoveryCmd.PersistentFlags().StringVarP(
		&cidr, "cidr", "c", cidr, `The CIDR of the subnet to automatically discover machines from.
		Such as: "172.17.1.0/24"`,
	)
	// Remove
	RemoveCmd.AddCommand(RemoveAutoDiscoveryCmd)
	RemoveAutoDiscoveryCmd.PersistentFlags().StringVarP(
		&name, "name", "n", name, "The name of the machine to remove.",
	)
	RemoveAutoDiscoveryCmd.PersistentFlags().BoolVarP(
		&yes, "yes", "y", yes, "Disable confirm prompt on deletion.",
	)
}
