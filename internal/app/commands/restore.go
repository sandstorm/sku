// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
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

package commands

import (
	"github.com/sandstorm/sku/internal/app/commands/restore"
	"github.com/spf13/cobra"
)

var restoreCommand = &cobra.Command{
	Use:   "restore",
	Short: "Restore a Kubernetes Namespace from a backup (manifests, data, SQL)",
	Long: `
See sub-commands for details.
`,
	Example: `
`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	RootCmd.AddCommand(restoreCommand)
	restoreCommand.AddCommand(restore.BuildCleanManifestsCommand())
	restoreCommand.AddCommand(restore.BuildMariadbCommand())
	restoreCommand.AddCommand(restore.BuildPersistentVolumesCommand())
}
