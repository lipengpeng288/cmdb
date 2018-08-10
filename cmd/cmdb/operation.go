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
	cobra "github.com/spf13/cobra"
)

// CreateCmd represents the create command of cmdb
var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create machine or digest",
	Long:  `Create machine or digest. Digest will be created on the backend.`,
}

// GetCmd represents the get command of cmdb
var GetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get machine or digest report",
	Long:  `Get machine or digest report.`,
}

// UpdateCmd represents the update command of cmdb
var UpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update machine information",
	Long:  `Update machine information. Some extra information are overidable to user.`,
}

// RemoveCmd represents the remove command of cmdb
var RemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove machine",
	Long:  `Remove machine`,
}

func init() {
	RootCmd.AddCommand(CreateCmd)
	RootCmd.AddCommand(GetCmd)
	RootCmd.AddCommand(UpdateCmd)
	RootCmd.AddCommand(RemoveCmd)
}
