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
	"os"
	"syscall"

	"github.com/logrusorgru/aurora"
	"github.com/sandstorm/sku/pkg/kubernetes"
	"github.com/sandstorm/sku/pkg/utility"
	"github.com/spf13/cobra"
	clientV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

var debugNoChroot bool
var debugImage string

// debugCmd for debugging into a conainer
var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Enter an interactive DEBUG / Root shell in a Kubernetes container",
	Long: `
Enter an interactive shell in a pod of the current namespace.
To select the pods you want to enter, you'll see a choice list.

Optionally, you can restrict the pod list by specifying a label
selector.

`,
	Example: `
# get presented a choice list which container to enter
	sku debug

# you can optionally specify a label selector to enter only a subset of pods
# You cannot specify a pod name directly, as they change very often anyways.
	sku debug app=foo
	sku debug app=foo,component=app

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
		} else {
			// use only existing container automatically
			containerName = podList.Items[i].Spec.Containers[0].Name
		}

		fmt.Printf("Connecting to %v %s in %v:\n", aurora.Green(podList.Items[i].Name), containerName, aurora.Green(currentContext))

		kubectlArgs := []string{"kubectl", "debug", podList.Items[i].Name, "-it", "--profile=general", fmt.Sprintf("--image=%s", debugImage), "--target", containerName}
		if !debugNoChroot {
			// we want to chroot
			kubectlArgs = append(kubectlArgs, "--", "chroot", "/proc/1/root")
		}

		syscall.Exec("/usr/local/bin/kubectl", kubectlArgs, os.Environ())
	},
}

func init() {
	RootCmd.AddCommand(debugCmd)
	debugCmd.Flags().BoolVarP(&debugNoChroot, "no-chroot", "", false, "if FALSE, you directly end up in the debug container; NOT in the target container. You find the target container in /proc/1/root.")
	debugCmd.Flags().StringVarP(&debugImage, "image", "", "busybox", "which image to use for debugging, e.g. nicolaka/netshoot. Usually you'll want to set --no-chroot then")
}
