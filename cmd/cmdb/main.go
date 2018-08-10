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
	client "github.com/universonic/cmdb/client"
)

// RootCmd represents the root command of cmdbd
var RootCmd = &cobra.Command{
	Use:   "cmdb",
	Short: "CMDB Client",
	Long: `CMDB Client Utility
-----------------------------------------------------
`,
}

var config = client.NewConfig()

func init() {
	RootCmd.PersistentFlags().StringVarP(
		&config.UnixAddr, "socket", "s", "/var/run/cmdb.sock", `Unix socket of the CMDB server endpoint.`,
	)
	RootCmd.PersistentFlags().StringVarP(
		&config.TCPAddr, "endpoint", "e", config.TCPAddr, `The HTTP endpoint of the CMDB server. It is a pair of value in 
colon-seperated IP address and port number. (format: <IPADDR:PORT>)`,
	)
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
