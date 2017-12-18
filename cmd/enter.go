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
	"github.com/spf13/cobra"
	"github.com/sandstorm/sku/utility"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	clientV1 "k8s.io/client-go/pkg/api/v1"
	"syscall"
	"os"
	"fmt"
	"github.com/logrusorgru/aurora"
	"bufio"
	"strconv"
	"strings"
)

// enterCmd represents the enter command
var enterCmd = &cobra.Command{
	Use:   "enter",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		currentContext := utility.KubernetesApiConfig().CurrentContext
		context := utility.KubernetesApiConfig().Contexts[currentContext]
		podList, _ := utility.KubernetesClientset().Pods(context.Namespace).List(v1.ListOptions{})

		for i, pod := range podList.Items {
			if pod.Status.Phase == clientV1.PodRunning {
				fmt.Printf("%d: %v - %v \n", i, aurora.Green(pod.Name), pod.Labels)
			} else {
				fmt.Printf("%d: %v - %v \n", i, pod.Name, pod.Labels)
			}

		}

		i := getNumberChoice()

		fmt.Printf("Connecting to %v in %v:\n", aurora.Green(podList.Items[i].Name), aurora.Green(currentContext))

		syscall.Exec("/usr/local/bin/kubectl", []string{"kubectl", "exec", "-it", podList.Items[i].Name, "/bin/bash"}, os.Environ())
	},
}

func getNumberChoice() int {
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Your Choice: ")
		userInput, _ := reader.ReadString('\n')
		i, err := strconv.Atoi(strings.TrimSpace(userInput))
		if err == nil {
			return i
		}
	}
}

func init() {
	rootCmd.AddCommand(enterCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// enterCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// enterCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
