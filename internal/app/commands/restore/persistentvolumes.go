package restore

import (
	"context"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/logrusorgru/aurora"
	"github.com/manifoldco/promptui"
	"github.com/sandstorm/sku/pkg/kubernetes"
	"github.com/sandstorm/sku/pkg/utility/wrapexec"
	"github.com/spf13/cobra"
	clientV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func BuildPersistentVolumesCommand() *cobra.Command {
	restoreBackupPath := ""

	persistentVolumesCommand := &cobra.Command{
		Use:   "persistentvolumes",
		Short: "Restore PersistentVolumes",
		Long: `
`,
		Example: `
`,

		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// we wrap the full code in a closure, and directly execute it,
			// so that we can run defer statements and they get executed before exiting
			resultCode := (func() int {
				//=================================
				// Preparation (Parameter Parsing)
				//=================================
				fmt.Println(aurora.Bold("Restoring Persistent Volumes to the Kubernetes cluster"))
				fmt.Println(aurora.Bold("======================================================"))
				fmt.Println("This is an interactive wizard guiding you through restoring Persistent Volumes.")
				fmt.Println("Before any destructive operation, you'll be asked whether you want to continue.")
				fmt.Println("")
				fmt.Println("")

				persistentVolumesBackupFolder := args[0]
				if len(persistentVolumesBackupFolder) == 0 {
					fmt.Printf("%s the persistent volumes folder must be given as parameter\n", aurora.Red("ERROR:"))
					return 1
				}

				fileStats, err := os.Stat(persistentVolumesBackupFolder)
				if err != nil {
					fmt.Printf("%s persistent volumes folder %s not found:\n    %s\n", aurora.Red("ERROR:"), aurora.Bold(persistentVolumesBackupFolder), err)
					return 1
				}
				if !fileStats.IsDir() {
					fmt.Printf("%s persistent volumes folder %s is a file, but needs to be a directory\n", aurora.Red("ERROR:"), aurora.Bold(persistentVolumesBackupFolder))
					return 1
				}

				currentContext := kubernetes.KubernetesApiConfig().CurrentContext
				k8sContextDefinition := kubernetes.KubernetesApiConfig().Contexts[currentContext]

				fmt.Printf("1) K8S namespace %s in context %s\n", aurora.Green(k8sContextDefinition.Namespace), aurora.Green(currentContext))
				fmt.Println("")
				fmt.Println("   We will connect to the Persistent Volumes via a running Pod.")
				fmt.Println("")
				fmt.Println()
				podName := kubernetes.SelectPod("Please select a Pod whose persistent volumes to restore")

				// query for running pods in current namespace
				pod, _ := kubernetes.KubernetesClientset().CoreV1().Pods(k8sContextDefinition.Namespace).Get(context.Background(), podName, metav1.GetOptions{})

				restoreBackupPath = path.Join(restoreBackupPath, time.Now().Format("01-02-2006-15-04-05")+"__"+k8sContextDefinition.Namespace)
				if err = os.MkdirAll(restoreBackupPath, os.ModePerm); err != nil {
					fmt.Printf("%s could not create %s:\n    %v\n", aurora.Red("ERROR:"), restoreBackupPath, err)
				}

				// we first iterate over the volumes, as we want to only restore each volume once,
				// even if it is mounted in multiple containers.
				for _, volume := range pod.Spec.Volumes {
					if volume.PersistentVolumeClaim != nil && len(volume.PersistentVolumeClaim.ClaimName) > 0 {
						// we continue only for persistent volume claims, not for secret volumes (or other volume types)

						container, volumeMount, found := findFirstContainerMountingVolume(pod.Spec.Containers, &volume)
						if !found {
							fmt.Println(aurora.Yellow(fmt.Sprintf("WARNING: Did not find mount point for Volume %s\n", volume.Name)))
							fmt.Println("")
							fmt.Println("   This means we cannot restore this volume, as it is not mounted in the given container.")
							fmt.Println("   You can check if another Pod is mounting this volume, then re-run this command and select the other Pod.")
							fmt.Println("")
							fmt.Println("Continuing with next volume now.")
							continue
						}

						persistentVolumesBackupFolders := buildFileListToRead(persistentVolumesBackupFolder, func(fileName string) bool {
							return true
						})

						prompt := promptui.Select{
							Label: aurora.Bold(fmt.Sprintf("Which backup should be replayed at %s?", volumeMount.MountPath)),
							Items: persistentVolumesBackupFolders,
						}
						_, chosenPersistentVolumesBackup, err := prompt.Run()

						command, err := wrapexec.RunWrappedCommand(
							"    [kubectl cp] ",
							"kubectl",
							"cp",
							fmt.Sprintf("%s:%s", podName, volumeMount.MountPath),
							fmt.Sprintf("%s", filepath.Join(restoreBackupPath, fmt.Sprintf("%s__%s", volume.Name, strings.ReplaceAll(volumeMount.MountPath, "/", "_")))),
							"-c",
							container.Name,
						)
						if err != nil {
							fmt.Printf("%s could not download persistent volume contents:\n    Command: %s\n    Error: %v\n", aurora.Red("ERROR:"), command.String(), err)
							return 1
						}

						confirmationPrompt := promptui.Prompt{
							Label:     aurora.Bold("Clear the persistent volume and restore its backup?"),
							IsConfirm: true,
						}
						_, err = confirmationPrompt.Run()
						if err != nil {
							fmt.Printf("user aborted.\n")
							return 1
						}

						// TODO: delete dotfiles "." as well
						command, err = wrapexec.RunWrappedCommand(
							"    [kubectl exec] ",
							"kubectl",
							"exec",
							podName,
							"-c",
							container.Name,
							"--",
							"/bin/sh",
							"-c",
							fmt.Sprintf("rm -Rf %s/*", volumeMount.MountPath),
						)
						if err != nil {
							fmt.Printf("%s could not clear persistent volume contents:\n    Command: %s\n    Error: %v\n", aurora.Red("ERROR:"), command.String(), err)
							return 1
						}

						command, err = wrapexec.RunWrappedCommand(
							"    [kubectl cp] ",
							"/bin/bash",
							"-c",
							fmt.Sprintf(
								"tar cf - -C %s . | kubectl exec -i --container=%s %s -- tar xf - -C %s",
								chosenPersistentVolumesBackup,
								container.Name,
								podName,
								volumeMount.MountPath,
							),
						)
						if err != nil {
							fmt.Printf("%s could not restore persistent volume contents:\n    Command: %s\n    Error: %v\n", aurora.Red("ERROR:"), command.String(), err)
							return 1
						}

					}
				}

				return 0
			})()
			os.Exit(resultCode)
		},
	}

	userHomeDir, _ := os.UserHomeDir()
	persistentVolumesCommand.Flags().StringVarP(&restoreBackupPath, "restoreBackupPath", "", filepath.Join(userHomeDir, "src/k8s/restore-backups"), "filename that contains the configuration to apply")

	return persistentVolumesCommand
}

func findFirstContainerMountingVolume(containers []clientV1.Container, volume *clientV1.Volume) (clientV1.Container, clientV1.VolumeMount, bool) {
	for _, container := range containers {
		for _, volumeMount := range container.VolumeMounts {
			if volumeMount.Name == volume.Name {
				return container, volumeMount, true
			}
		}
	}
	return clientV1.Container{}, clientV1.VolumeMount{}, false
}
