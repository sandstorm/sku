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
	"fmt"
	"github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
)

var addConfigCommand = &cobra.Command{
	Use:   "add-config",
	Short: "Add an external kubeconfig file to ~/.kube/config (i.e. merge the contents together)",
	Long: `
This command allows adding an external kubeconfig file to the default ~/.kube/config.
so you can actually merge multiple Kubernetes configs together which you got from different sources.

It is basically doing what is explained in https://github.com/kubernetes/kubernetes/issues/46381#issuecomment-461404505
`,
	Example: `
	sku add-config path-to-additional-kubeconfig-file
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		otherKubeconfigFile := args[0]

		env := os.Environ()
		userHomeDir, _ := os.UserHomeDir()
		kubeConfig := userHomeDir + "/.kube/config"

		kubectlCommand := exec.Command("/usr/local/bin/kubectl", "config", "view", "--flatten")
		kubectlCommand.Env = append(env, fmt.Sprintf(`KUBECONFIG=%s:%s`, kubeConfig, otherKubeconfigFile))
		output, err := kubectlCommand.Output()
		if err != nil {
			log.Fatal(err)
		}

		ioutil.WriteFile(kubeConfig, output, 0644)

		fmt.Printf("%v\n", aurora.Green("Updated ~/.kube/config."))
	},
}


func init() {
	RootCmd.AddCommand(addConfigCommand)
}
