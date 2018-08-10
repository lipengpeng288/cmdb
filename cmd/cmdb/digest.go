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

// CreateDigestCmd represents the digest command of cmdb
var CreateDigestCmd = &cobra.Command{
	Use:   "digest",
	Short: "Trigger a digest update immediately",
	Long:  `Trigger a digest update immediately.`,
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
		err = cli.CreateMachineDigest()
		if err != nil {
			panic(err)
		}
	},
}

// GetDigestCmd represents the digest command of cmdb
var GetDigestCmd = &cobra.Command{
	Use:   "digest",
	Short: "Generate a digest report from backend.",
	Long: `Generate a digest report from backend. Omit "--name" and "--date"
to generate a daily report.`,
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
		if list {
			err = cli.ListMachineDigest(args...)
			if err != nil {
				panic(err)
			}
			return
		}
		err = cli.GetMachineDigest(name, dateStr, output, jsoned)
		if err != nil {
			panic(err)
		}
	},
}

var (
	dateStr, output string
	jsoned, list    bool
)

func init() {
	CreateCmd.AddCommand(CreateDigestCmd)
	GetCmd.AddCommand(GetDigestCmd)

	GetDigestCmd.PersistentFlags().StringVarP(
		&name, "name", "n", name, `The name of the machine to generate report for.
It cannot be use with "--date".`,
	)
	GetDigestCmd.PersistentFlags().StringVarP(
		&dateStr, "date", "d", dateStr, `The date that you want to generate report from. The 
report data will be chosen from the latest chunk from that date.
Format: RFC3339 ("2006-01-02T15:04:05Z07:00")`,
	)
	GetDigestCmd.PersistentFlags().StringVarP(
		&output, "output", "o", output, `The file to write result to. It must not be existed.`,
	)
	GetDigestCmd.PersistentFlags().BoolVar(
		&jsoned, "json", jsoned, `Whether to enforce to use JSON format in report.`,
	)
	GetDigestCmd.PersistentFlags().BoolVar(
		&list, "list", list, `List all existing digest.`,
	)
}
