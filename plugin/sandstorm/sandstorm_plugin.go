package main

import (
	"fmt"
	"github.com/sandstorm/sku/pkg/kubernetes"
	"github.com/spf13/cobra"
	"io/ioutil"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"os"
	"os/signal"
	"syscall"
)

type plugin struct {
}

const backupSecretName = "read-from-backup"

func main() {
	println("This is a sku plugin - not to be executed directly. This main function is needed to make goreleaser happy :-)")
}

var downloadCommand = &cobra.Command{
	Use:   "downloadData",
	Short: "ALPHA: (sandstorm) Download data from the backup",
	Long: `
ALPHA Quality!

List and switch kubernetes contexts.`,
	Example: `
# List all kubernetes contexts:
	sku context

# Switch to a kubernetes context:
	sku context [contextName]
`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		namespaceList, _ := kubernetes.KubernetesClientset().CoreV1().Namespaces().List(meta_v1.ListOptions{})

		if len(args) == 0 {
			fmt.Printf("Namespace: \n")
			kubernetes.PrintExistingNamespaces(namespaceList)
		} else {
			namespace := args[0]
			kubernetes.EnsureNamespaceExists(namespace, namespaceList)

			secret, err := kubernetes.KubernetesClientset().CoreV1().Secrets(namespace).Get(backupSecretName, meta_v1.GetOptions{})
			if err != nil {
				log.Fatalf("Secrets could not be fetched: %s", err)
			}

			//googleProjectId := secret.Data["GOOGLE_PROJECT_ID"]
			//resticPassword := secret.Data["RESTIC_PASSWORD"]
			googleServiceAccountJsonKey := secret.Data["GOOGLE_SERVICE_ACCOUNT_JSON_KEY"]

			googleServiceKeyTempFile, err := ioutil.TempFile("", "restic_key")
			if err != nil {
				log.Fatalf("Google Service Key Temp file could not be created: %s", err)
			}
			// clean up
			defer os.Remove(googleServiceKeyTempFile.Name())
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigs
				os.Remove(googleServiceKeyTempFile.Name())
				os.Exit(0)
			}()

			if _, err := googleServiceKeyTempFile.Write(googleServiceAccountJsonKey); err != nil {
				log.Fatal(err)
			}
			if err := googleServiceKeyTempFile.Close(); err != nil {
				log.Fatal(err)
			}

		}
	},
}

func (p plugin) InitializeCommands(rootCommand *cobra.Command) {
	rootCommand.AddCommand(downloadCommand)
}

var Plugin plugin
