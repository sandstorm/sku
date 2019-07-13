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

	"github.com/spf13/cobra"
	"github.com/logrusorgru/aurora"
	"k8s.io/client-go/tools/clientcmd/api"
	"log"
	"k8s.io/client-go/tools/clientcmd"
	"encoding/json"
	"os"
	"github.com/sandstorm/sku/internal/pkg/encryption"
	"github.com/sandstorm/sku/pkg/kubernetes"
	"github.com/sandstorm/sku/pkg/utility"
)

type EncryptedContainer struct {
	RsaEncryptedAesKey []byte
	AesEncryptedClientKeyData []byte
	AesEncryptedClientCertificateData []byte
}

var encryptKubernetesCredentials = &cobra.Command{
	Use:   "encrypt",
	Short: "ALPHA: Encrypt Kubernetes credentials via Yubikey PIV module",
	Long: `
ALPHA: Encrypt Kubernetes credentials via Yubikey PIV module

You can encrypt the Client Credentials using your Yubikey's private key. This command 
encrypts the keys and changes the Kubernetes config to decrypt the keys when you need them,
by setting up the "sku decryptCredentials" command as Exec Authentication Plugin for Kubectl.

As a user, this means that you always need to touch your Yubikey when issuing kubectl commands.

PREREQUISITES:
- install https://github.com/sandstorm/ykpiv-ssh-agent-helper in a recent version OR install
  OpenSC from https://github.com/OpenSC/OpenSC/releases
- ensure kubectl is installed in at least version 1.11.
`,
	Example: `
# set up encryption for the given context
# You need to tap your yubikey during setup; so that decryption can be properly tested.
	sku encrypt [context]
`,

	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		kubernetes.EnsureVersionOfKubernetesCliSupportsExternalAuth()
		encryption.SetupCrypto()

		contextToSetupEncryptionFor := args[0]
		kubernetes.EnsureContextExists(contextToSetupEncryptionFor)

		fmt.Printf("Setting up encryption for context %v\n", aurora.Green(contextToSetupEncryptionFor))
		config := kubernetes.KubernetesApiConfig()
		context := config.Contexts[contextToSetupEncryptionFor]
		fmt.Printf("- found user %v\n", aurora.Bold(context.AuthInfo))
		authInfo := config.AuthInfos[context.AuthInfo]

		aesKey := encryption.GenerateRandomAesKey()

		// TODO: implement case that ClientKey is specified as file.
		if len(authInfo.ClientKeyData) == 0 {
			log.Fatalln("FATAL: ClientKeyData was empty or null; we do not support external token paths yet.")
		}

		if len(authInfo.ClientCertificateData) == 0 {
			log.Fatalln("FATAL: ClientCertificateData was empty or null; we do not support external token paths yet.")
		}

		container := &EncryptedContainer {
			RsaEncryptedAesKey: encryption.EncryptAesKeyViaYubikey(aesKey),
			AesEncryptedClientKeyData: encryption.EncryptAes(aesKey, authInfo.ClientKeyData),
			AesEncryptedClientCertificateData: encryption.EncryptAes(aesKey, authInfo.ClientCertificateData),
		}

		jsonContainer, err := json.Marshal(container)
		if err != nil {
			log.Fatalf("Error: container could not be serialized: %s\n", err)
		}

		authInfo.Exec = &api.ExecConfig{
			Command: utility.GetSkuExecutableFileName(),
			Args:[]string{"decryptCredentials", context.AuthInfo},
			APIVersion: "client.authentication.k8s.io/v1beta1",
			Env: []api.ExecEnvVar{
				{
					Name:  "encrypted-client-key-and-certificate-data",
					Value: string(jsonContainer),
				},
			},
		}
		authInfo.ClientCertificateData = nil
		authInfo.ClientKeyData = nil

		clientcmd.ModifyConfig(clientcmd.NewDefaultPathOptions(), *config, false)
	},
}

type execCredential struct {
	ApiVersion string `json:"apiVersion"`
	Kind string `json:"kind"`
	Status execCredentialStatus `json:"status"`
}

type execCredentialStatus struct {
	ClientCertificateData string `json:"clientCertificateData"`
	ClientKeyData string `json:"clientKeyData"`
}

var decryptKubernetesCredentials = &cobra.Command{
	Use:   "decryptCredentials",
	Hidden: true,
	Short: "ALPHA: Decrypt Kubernetes credentials (part of sku encrypt)",
	Long: `
You never need to run this command manually, as "sku encrypt" sets up this command as external kubectl
auth provider.
`,

	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		encryption.SetupCrypto()

		authInfoToDecrypt := args[0]
		authInfo := kubernetes.KubernetesApiConfig().AuthInfos[authInfoToDecrypt]

		jsonContainer := findEncryptedClientKeyAndCertificateInAuthInfo(authInfo)
		container := &EncryptedContainer {}
		err := json.Unmarshal([]byte(jsonContainer), &container)
		if err != nil {
			log.Fatalf("There as an error unmarshalling kube credentials: %v\n", err)
		}

		fmt.Fprintf(os.Stderr, "Please tap your yubikey TWICE to decrypt the Kubernetes credentials\n")

		aesKey := encryption.DecryptAesKeyViaYubikey(container.RsaEncryptedAesKey)
		clientCertificateData := encryption.DecryptAes(aesKey, container.AesEncryptedClientCertificateData)
		clientKeyData := encryption.DecryptAes(aesKey, container.AesEncryptedClientKeyData)

		result := &execCredential{
			ApiVersion: "client.authentication.k8s.io/v1beta1",
			Kind: "ExecCredential",
			Status: execCredentialStatus{
				ClientCertificateData: string(clientCertificateData),
				ClientKeyData: string(clientKeyData),
			},
		}

		resultString, err := json.Marshal(result)
		if err != nil {
			log.Fatalf("There as an error marshalling the ExecCredential: %v\n", err)
		}

		fmt.Println(string(resultString))
	},
}
func findEncryptedClientKeyAndCertificateInAuthInfo(authInfo *api.AuthInfo) string {
	for _, element := range authInfo.Exec.Env {
		if element.Name == "encrypted-client-key-and-certificate-data" {
			return element.Value
		}
	}
	log.Fatalf("!!! did not find encrypted client key in users section of kubeconfig file. ABORTING.\n")
	return ""
}



func init() {
	RootCmd.AddCommand(encryptKubernetesCredentials)
	RootCmd.AddCommand(decryptKubernetesCredentials)
}
