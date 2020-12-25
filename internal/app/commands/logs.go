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
	"context"
	"fmt"
	"github.com/logrusorgru/aurora"
	"github.com/sandstorm/sku/pkg/kubernetes"
	"github.com/spf13/cobra"
	clientV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"syscall"
)

// logsCmd represents the enter command
var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show logs in a Kubernetes container",
	Long: `
Show the logs of a pod of the current namespace.
To select the pods you want to get the logs for, you'll see a choice list.

Optionally, you can restrict the pod list by specifying a label
selector.

`,
	Example: `
# get presented a choice list which logs to show
	sku logs

# you can optionally specify a label selector to show only the specific logs
# You cannot specify a pod name directly, as they change very often anyways.
	sku logs app=foo
	sku logs app=foo,component=app

`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		currentContext := kubernetes.KubernetesApiConfig().CurrentContext
		k8sContextDefinition := kubernetes.KubernetesApiConfig().Contexts[currentContext]
		labelSelector := ""
		if len(args) == 1 {
			labelSelector = args[0]
			fmt.Printf("Listing pods with label %v in namespace %v in k8sContextDefinition %v.\n", aurora.Green(labelSelector), aurora.Green(k8sContextDefinition.Namespace), aurora.Green(currentContext))
		} else {
			fmt.Printf("Listing pods in namespace %v in k8sContextDefinition %v.\n", aurora.Green(k8sContextDefinition.Namespace), aurora.Green(currentContext))
		}

		podList, _ := kubernetes.KubernetesClientset().CoreV1().Pods(k8sContextDefinition.Namespace).List(context.Background(), v1.ListOptions{
			LabelSelector: labelSelector,
		})

		for i, pod := range podList.Items {
			if pod.Status.Phase == clientV1.PodRunning {
				fmt.Printf("%d: %v - %v \n", i, aurora.Green(pod.Name), pod.Labels)
			} else {
				fmt.Printf("%d: %v - %v \n", i, pod.Name, pod.Labels)
			}
		}

		i := getNumberChoice()

		fmt.Printf("Showing Logs to %v in %v:\n", aurora.Green(podList.Items[i].Name), aurora.Green(currentContext))

		syscall.Exec("/usr/local/bin/kubectl", []string{"kubectl", "logs", "-f", podList.Items[i].Name}, os.Environ())
	},
}

func init() {
	RootCmd.AddCommand(logsCmd)
}
