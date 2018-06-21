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

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sandstorm/sku/utility"
	"github.com/logrusorgru/aurora"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

// contextCmd represents the context command
var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Switch the Kubernetes Context",
	Long: `
List and switch kubernetes contexts.`,
	Example: `
# List all kubernetes contexts:
	sku context

# Switch to a kubernetes context:
	sku context [contextName]
`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Printf("Contexts: \n")
			printExistingContexts()
		} else {
			config := utility.KubernetesApiConfig()
			newContext := args[0]

			foundContext := false
			for context := range utility.KubernetesApiConfig().Contexts {
				if context == newContext {
					foundContext = true
				}
			}

			if !foundContext {
				fmt.Printf("%v\n", aurora.Red("Context not found; use one of the list below:"))
				printExistingContexts()
				os.Exit(1)
			}

			config.CurrentContext = newContext
			clientcmd.ModifyConfig(clientcmd.NewDefaultPathOptions(), *config, false)

			fmt.Printf("Switched to context %v.\n", aurora.Green(newContext))
		}
	},
}

func printExistingContexts() {
	currentContext := utility.KubernetesApiConfig().CurrentContext
	for context := range utility.KubernetesApiConfig().Contexts {
		if context == currentContext {
			fmt.Printf("* %v\n", aurora.Green(context))
		} else {
			fmt.Printf("  %v\n", context)
		}
	}

}

func init() {
	rootCmd.AddCommand(contextCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// contextCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// contextCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
