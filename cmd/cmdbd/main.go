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
	server "github.com/universonic/cmdb/server"
	fsutil "github.com/universonic/cmdb/utils/filesystem"
)

// RootCmd represents the root command of cmdbd
var RootCmd = &cobra.Command{
	Use:   "cmdbd",
	Short: "CMDB Daemon",
	Long: `CMDB Daemon Utility
-----------------------------------------------------
`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		yes, err := fsutil.FileExists(configFile)
		if !yes {
			return fmt.Errorf("File not found: %s", configFile)
		}
		return err
	},
	Run: func(cmd *cobra.Command, args []string) {
		s, err := server.ParseFromFile(configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(2)
		}
		if err = s.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(3)
		}
	},
}

var configFile string

func init() {
	RootCmd.PersistentFlags().StringVarP(
		&configFile, "config", "c", "/etc/cmdb/cmdb.toml", `The configuration file of CMDB Daemon.
Supported format: TOML (default), YAML, and JSON.`,
	)
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
