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

package main

import (
	"fmt"
	"os"

	cobra "github.com/spf13/cobra"
)

// CreateMachineCmd represents the machine command of cmdb
var CreateMachineCmd = &cobra.Command{
	Use:   "machine",
	Short: "Create a machine from prompt or YAML.",
	Long:  `Create a machine from prompt or YAML. If both specified, will use YAML.`,
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
		if yamlFile != "" {
			err = cli.CreateMachineFromYAML(yamlFile, yamled)
			if err != nil {
				panic(err)
			}
			return
		}
		err = cli.CreateMachine(name, sshAddr, sshPort, sshUser, ipmiAddr, ipmiUser, ipmiPassword, yamled)
		if err != nil {
			panic(err)
		}
	},
}

// GetMachineCmd represents the machine command of cmdb
var GetMachineCmd = &cobra.Command{
	Use:   "machine",
	Short: "Overview machine configuration",
	Long:  `Overview machine configuration. Possible to query multiple machines at one time.`,
	Run: func(cmd *cobra.Command, args []string) {
		var machines []string
		if all {
			machines = []string{"*"}
		} else if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "At least one machine name must be given.\n")
			os.Exit(1)
		} else {
			machines = args
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
		err = cli.GetMachine(machines, yamled)
		if err != nil {
			panic(err)
		}
	},
}

// UpdateMachineCmd represents the machine command of cmdb
var UpdateMachineCmd = &cobra.Command{
	Use:   "machine",
	Short: "Update a machine from prompt or YAML.",
	Long:  `Update a machine from prompt or YAML. If both specified, will use YAML.`,
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
		if yamlFile != "" {
			err = cli.UpdateMachineFromYAML(yamlFile, yamled)
			if err != nil {
				panic(err)
			}
			return
		}
		err = cli.UpdateMachine(name, sshAddr, sshPort, sshUser, ipmiAddr, ipmiUser, ipmiPassword, yamled)
		if err != nil {
			panic(err)
		}
	},
}

// RemoveMachineCmd represents the machine command of cmdb
var RemoveMachineCmd = &cobra.Command{
	Use:   "machine",
	Short: "Remove a machine",
	Long:  `Remove a machine`,
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
		err = cli.RemoveMachine(name, yes)
		if err != nil {
			panic(err)
		}
	},
}

var (
	yamlFile                                  string
	name, sshAddr                             string
	sshPort                                   uint16
	sshUser, ipmiAddr, ipmiUser, ipmiPassword string
	yamled, all, yes                          bool
)

func init() {
	// Create
	CreateCmd.AddCommand(CreateMachineCmd)
	CreateMachineCmd.PersistentFlags().StringVarP(
		&yamlFile, "from-yaml", "F", yamlFile, "The YAML file to use for creating machine.",
	)
	CreateMachineCmd.PersistentFlags().BoolVar(
		&yamled, "yaml", yamled, "Whether to use YAML format output rather than a ASCII table.",
	)
	CreateMachineCmd.PersistentFlags().StringVarP(
		&name, "name", "n", name, "The name of the machine to create.",
	)
	CreateMachineCmd.PersistentFlags().StringVarP(
		&sshAddr, "ssh-address", "H", name, "The IP address that the SSH server listens.",
	)
	CreateMachineCmd.PersistentFlags().Uint16VarP(
		&sshPort, "ssh-port", "P", 22, "The port of the SSH service.",
	)
	CreateMachineCmd.PersistentFlags().StringVar(
		&sshUser, "ssh-user", "root", "The login user of the SSH service.",
	)
	CreateMachineCmd.PersistentFlags().StringVarP(
		&ipmiAddr, "ipmi-address", "I", ipmiAddr, "The IP endpoint of the IPMI interface.",
	)
	CreateMachineCmd.PersistentFlags().StringVarP(
		&ipmiUser, "ipmi-user", "u", ipmiUser, "The login user of the IPMI interface.",
	)
	CreateMachineCmd.PersistentFlags().StringVarP(
		&ipmiPassword, "ipmi-password", "p", ipmiPassword, "The login password of the IPMI interface.",
	)
	// Get
	GetCmd.AddCommand(GetMachineCmd)
	GetMachineCmd.PersistentFlags().BoolVar(
		&all, "all", all, "Select all existing machine.",
	)
	GetMachineCmd.PersistentFlags().BoolVar(
		&yamled, "yaml", yamled, "Whether to use YAML format output rather than a ASCII table.",
	)
	// Update
	UpdateCmd.AddCommand(UpdateMachineCmd)
	UpdateMachineCmd.PersistentFlags().StringVarP(
		&yamlFile, "from-yaml", "F", yamlFile, "The YAML file to use for updating machine.",
	)
	UpdateMachineCmd.PersistentFlags().BoolVar(
		&yamled, "yaml", yamled, "Whether to use YAML format output rather than a ASCII table.",
	)
	UpdateMachineCmd.PersistentFlags().StringVarP(
		&name, "name", "n", name, "The name of the machine to update.",
	)
	UpdateMachineCmd.PersistentFlags().StringVarP(
		&sshAddr, "ssh-address", "H", name, "The IP address that the SSH server listens.",
	)
	UpdateMachineCmd.PersistentFlags().Uint16VarP(
		&sshPort, "ssh-port", "P", 22, "The port of the SSH service.",
	)
	UpdateMachineCmd.PersistentFlags().StringVar(
		&sshUser, "ssh-user", "root", "The login user of the SSH service.",
	)
	UpdateMachineCmd.PersistentFlags().StringVarP(
		&ipmiAddr, "ipmi-address", "I", ipmiAddr, "The IP endpoint of the IPMI interface.",
	)
	UpdateMachineCmd.PersistentFlags().StringVarP(
		&ipmiUser, "ipmi-user", "u", ipmiUser, "The login user of the IPMI interface.",
	)
	UpdateMachineCmd.PersistentFlags().StringVarP(
		&ipmiPassword, "ipmi-password", "p", ipmiPassword, "The login password of the IPMI interface.",
	)
	// Remove
	RemoveCmd.AddCommand(RemoveMachineCmd)
	RemoveMachineCmd.PersistentFlags().StringVarP(
		&name, "name", "n", name, "The name of the machine to remove.",
	)
	RemoveMachineCmd.PersistentFlags().BoolVarP(
		&yes, "yes", "y", yes, "Disable confirm prompt on deletion.",
	)
}
