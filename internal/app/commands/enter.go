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
	"github.com/sandstorm/sku/pkg/utility"
	"github.com/spf13/cobra"
	clientV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"syscall"
)

// enterCmd represents the enter command
var enterCmd = &cobra.Command{
	Use:   "enter",
	Short: "Enter an interactive shell in a Kubernetes container",
	Long: `
Enter an interactive shell in a pod of the current namespace.
To select the pods you want to enter, you'll see a choice list.

Optionally, you can restrict the pod list by specifying a label
selector.

`,
	Example: `
# get presented a choice list which container to enter
	sku enter

# you can optionally specify a label selector to enter only a subset of pods
# You cannot specify a pod name directly, as they change very often anyways.
	sku enter app=foo
	sku enter app=foo,component=app

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

		numberOfRunningPods := 0
		lastRunningPodIndex := 0
		for i, pod := range podList.Items {
			if pod.Status.Phase == clientV1.PodRunning {
				fmt.Printf("%d: %v - %v \n", i, aurora.Green(pod.Name), pod.Labels)
				numberOfRunningPods++
			} else {
				fmt.Printf("%d: %v - %v \n", i, pod.Name, pod.Labels)
			}
		}

		var i int
		switch numberOfRunningPods {
		case 0:
			fmt.Printf("No running pods. Exiting!\n")
			os.Exit(1)
		case 1:
			i = lastRunningPodIndex
		default:
			i = utility.GetNumberChoice()
		}

		containerName := ""
		if len(podList.Items[i].Spec.Containers) > 1 {
			fmt.Printf("Which container?.\n")
			for ci, c := range podList.Items[i].Spec.Containers {
				fmt.Printf("%d: %v\n", ci, aurora.Green(c.Name))
			}
			ci := utility.GetNumberChoice()

			containerName = podList.Items[i].Spec.Containers[ci].Name
		}

		fmt.Printf("Connecting to %v %s in %v:\n", aurora.Green(podList.Items[i].Name), containerName, aurora.Green(currentContext))

		kubectlArgs := []string{"kubectl", "exec", "-it"}
		if containerName != "" {
			kubectlArgs = append(kubectlArgs, "-c", containerName)
		}
		// enter bash or sh
		kubectlArgs = append(kubectlArgs, podList.Items[i].Name, "--", "/bin/sh", "-c", "[ -e /bin/bash ] && exec /bin/bash || exec /bin/sh")
		syscall.Exec("/usr/local/bin/kubectl", kubectlArgs, os.Environ())
	},
}

func init() {
	RootCmd.AddCommand(enterCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// enterCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// enterCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
