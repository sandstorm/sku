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
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/logrusorgru/aurora"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"k8s.io/api/core/v1"
)

// nsCmd represents the ns command
var nsCmd = &cobra.Command{
	Use:   "ns",
	Short: "Switch the Kubernetes Namespace",
	Long: `
List and switch kubernetes namespaces.`,
	Example: `
# List all kubernetes namespaces in the current context:
	sku ns

# Switch to a kubernetes namespace:
	sku ns [namespaceName]
`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		namespaceList, _ := utility.KubernetesClientset().CoreV1().Namespaces().List(meta_v1.ListOptions{})

		if len(args) == 0 {
			fmt.Printf("Namespaces: \n")
			printExistingNamespaces(namespaceList)
		} else {
			config := utility.KubernetesApiConfig()
			currentContext := config.CurrentContext
			context := utility.KubernetesApiConfig().Contexts[currentContext]

			newNamespace := args[0]

			foundNamespace := false
			for _, ns := range namespaceList.Items {
				if newNamespace == ns.Name {
					foundNamespace = true
				}
			}

			if !foundNamespace {
				fmt.Printf("%v\n", aurora.Red("Namespace not found; use one of the list below:"))
				printExistingNamespaces(namespaceList)
				os.Exit(1)
			}

			context.Namespace = newNamespace
			clientcmd.ModifyConfig(clientcmd.NewDefaultPathOptions(), *config, false)

			fmt.Printf("Switched to namespace %v in context %v.\n", aurora.Green(newNamespace), aurora.Green(currentContext))
		}
	},
}

func printExistingNamespaces(namespaceList *v1.NamespaceList) {
	currentContext := utility.KubernetesApiConfig().CurrentContext
	context := utility.KubernetesApiConfig().Contexts[currentContext]

	for _, ns := range namespaceList.Items {
		if context.Namespace == ns.Name {
			fmt.Printf("* %v\n", aurora.Green(ns.Name))
		} else {
			fmt.Printf("  %v\n", ns.Name)
		}

	}
}



func init() {
	rootCmd.AddCommand(nsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// nsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// nsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

