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

package kubernetes

import (
	"path/filepath"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/kubernetes"
	"os"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
	"bytes"
	"os/exec"
	"log"
	"encoding/json"
	"strconv"
	"fmt"
	"github.com/logrusorgru/aurora"
	"k8s.io/api/core/v1"
)

var clientset *kubernetes.Clientset
var config *rest.Config
var apiConfig *api.Config
func KubernetesInit() {
	var kubeconfig = filepath.Join(homeDir(), ".kube", "config")

	var err error
	apiConfig = clientcmd.GetConfigFromFileOrDie(kubeconfig)
	// use the current context in kubeconfig
	config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
}


func KubernetesConfig() *rest.Config {
	return config
}

func KubernetesApiConfig() *api.Config {
	return apiConfig
}


func KubernetesClientset() *kubernetes.Clientset {
	return clientset
}


func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

type VersionResponse struct {
	ClientVersion ClientVersionResponse `json:"clientVersion"`
}

type ClientVersionResponse struct {
	Major string `json:"major"`
	Minor string `json:"minor"`
}
func EnsureVersionOfKubernetesCliSupportsExternalAuth() {
	cmd := exec.Command("kubectl", "version", "--client", "--output=json")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	response := &VersionResponse{}
	json.Unmarshal(out.Bytes(), response)

	major, _ := strconv.Atoi(response.ClientVersion.Major)
	minor, _ := strconv.Atoi(response.ClientVersion.Minor)

	supportsExternalAuth := (major == 1 && minor >= 11) || major >= 2
	if !supportsExternalAuth {
		log.Printf("ERROR: kubectl must be at least version 1.11 to support External Auth.\n")
		log.Printf("Found version was: %d.%d.\n", major, minor)
		log.Printf("To fix this issue, run 'brew install kubernetes-cli' or 'brew upgrade kubernetes-cli'\n")
		log.Fatalf("ABORTING!\n")
	}
}


func EnsureContextExists(newContext string) {
	foundContext := false
	for context := range KubernetesApiConfig().Contexts {
		if context == newContext {
			foundContext = true
		}
	}

	if !foundContext {
		fmt.Printf("%v\n", aurora.Red("Context not found; use one of the list below:"))
		PrintExistingContexts()
		os.Exit(1)
	}

}

func PrintExistingContexts() {
	currentContext := KubernetesApiConfig().CurrentContext
	for context := range KubernetesApiConfig().Contexts {
		if context == currentContext {
			fmt.Printf("* %v\n", aurora.Green(context))
		} else {
			fmt.Printf("  %v\n", context)
		}
	}

}

func EnsureNamespaceExists(namespace string, namespaceList *v1.NamespaceList) {
	foundNamespace := false
	for _, ns := range namespaceList.Items {
		if namespace == ns.Name {
			foundNamespace = true
		}
	}

	if !foundNamespace {
		fmt.Printf("%v\n", aurora.Red("Namespace not found; use one of the list below:"))
		PrintExistingNamespaces(namespaceList)
		os.Exit(1)
	}
}

func PrintExistingNamespaces(namespaceList *v1.NamespaceList) {
	currentContext := KubernetesApiConfig().CurrentContext
	context := KubernetesApiConfig().Contexts[currentContext]

	for _, ns := range namespaceList.Items {
		if context.Namespace == ns.Name {
			fmt.Printf("* %v\n", aurora.Green(ns.Name))
		} else {
			fmt.Printf("  %v\n", ns.Name)
		}

	}
}
