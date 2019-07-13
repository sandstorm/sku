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
	"github.com/sandstorm/sku/internal/pkg/rancher"
	"github.com/spf13/cobra"
)

var apiEndpointUrl string
var token string
var outputDirectory string

var rancherBackupCommand = &cobra.Command{
	Use:   "rancher-backup",
	Short: "ALPHA: Backup a Rancher server by fetching all API resources",
	Long: `
This command allows backing up a Rancher server, by fetching all API resources. The result is stored
in the current directory, and should be versioned in Git.

NOTE: it is *NOT* possible to directly import the Rancher config which has been exported this way,
      but it can help to have a human-readable representation of the different resources; so that
      it is traceable when/if some options have changed.
`,
	Example: `
	sku rancher-backup --url https://your-rancher-server.de/v3 --token BEARER-TOKEN-HERE --output ./backup-directory
`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		rancher.RunBackup(apiEndpointUrl, token, outputDirectory)
	},
}

func init() {
	rancherBackupCommand.Flags().StringVarP(&apiEndpointUrl, "url", "u", "", "API URL of the rancher server")
	rancherBackupCommand.MarkFlagRequired("url")

	rancherBackupCommand.Flags().StringVarP(&token, "token", "t", "", "Bearer Token from Rancher")
	rancherBackupCommand.MarkFlagRequired("token")

	rancherBackupCommand.Flags().StringVarP(&outputDirectory, "output", "o", "", "Output directory")
	rancherBackupCommand.MarkFlagRequired("output")

	RootCmd.AddCommand(rancherBackupCommand)
}
