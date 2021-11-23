package commands

import (
	"context"
	"fmt"
	. "github.com/logrusorgru/aurora/v3"
	"github.com/sandstorm/sku/pkg/kubernetes"
	"github.com/spf13/cobra"
	"io/ioutil"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"syscall"
)

const backupSecretNamespace = "backup-job-downloads"
const backupSecretName = "backup-readonly-credentials"

func init() {
	RootCmd.AddCommand(mountCommand)
	RootCmd.AddCommand(umountCommand)
}

var mountCommand = &cobra.Command{
	Use:     "mount-backup",
	Aliases: []string{"mount"},
	Short:   "ALPHA: (sandstorm) Mount the backup to ~/src/k8s/backup/[nodename]",
	Long: `
ALPHA Quality!

Prerequisites (needed for mounting to work on OSX >=11.0):  
     - Install OSXFuse by downloading https://github.com/osxfuse/osxfuse/releases (for OSX >=11.0, version 4 is known to work).
     - Install Borgbackup by downloading borg-macos64 from https://github.com/borgbackup/borg/releases, placing it to /usr/local/bin/borg and make it executable.

On first run, you'll get an error about untrusted software. '
`,
	Example: `
	sku mount-backup k3s2021-1
	sku mount-backup k3s2021-2
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		kubernetesNode := args[0]

		fmt.Printf("Trying to mount backup for %s\n\n", Bold(kubernetesNode))

		secret, err := kubernetes.KubernetesClientset().CoreV1().Secrets(backupSecretNamespace).Get(context.Background(), backupSecretName, meta_v1.GetOptions{})

		if err != nil {
			log.Fatalf("secret not found: %s", err)
		}

		borgbackupSshKeyTempFile, err := ioutil.TempFile("", "borgbackup_read_ssh_key_tempfile")
		if err != nil {
			log.Fatalf("borgbackupSshKeyTempFile could not be created: %s", err)
		}
		// clean up on exit
		defer os.Remove(borgbackupSshKeyTempFile.Name())
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigs
			os.Remove(borgbackupSshKeyTempFile.Name())
			os.Exit(0)
		}()

		if len(secret.Data["id_rsa"]) == 0 {
			os.Remove(borgbackupSshKeyTempFile.Name())
			log.Fatal("id_rsa cold not be found")
		}

		if _, err := borgbackupSshKeyTempFile.Write(secret.Data["id_rsa"]); err != nil {
			os.Remove(borgbackupSshKeyTempFile.Name())
			log.Fatal(err)
		}
		if err := borgbackupSshKeyTempFile.Close(); err != nil {
			os.Remove(borgbackupSshKeyTempFile.Name())
			log.Fatal(err)
		}

		borgbackupRepoUrl := string(secret.Data["repo_url_"+kubernetesNode])
		if len(borgbackupRepoUrl) == 0 {
			os.Remove(borgbackupSshKeyTempFile.Name())
			log.Fatalf(Colorize("Repo URL for node %s not found", RedBg).String(), kubernetesNode)
		}

		userHomeDir, _ := os.UserHomeDir()
		backupMountDir := userHomeDir + "/src/k8s/backup/" + kubernetesNode
		os.MkdirAll(backupMountDir, os.ModePerm)

		fmt.Printf("In case you have your yubikey attached, %s\n\n", Colorize("you might need to touch it if it blinks.", BoldFm|BlinkFm))
		fmt.Printf(Colorize("!!! In a few seconds, you'll be asked to enter the decryption key.\n", YellowFg).String())
		fmt.Printf(Colorize("    Check your password manager and search for Borgbackup.\n", YellowFg).String())

		sysUser, err := user.Current()
		if err != nil {
			panic(err)
		}

		borgCommand := exec.Command("/usr/local/bin/borg", "mount", "-o", "uid="+sysUser.Uid, "--last", "1", "--strip-components", "1", borgbackupRepoUrl, backupMountDir)
		env := os.Environ()
		borgCommand.Env = append(env, fmt.Sprintf(`BORG_RSH=ssh -i %s`, borgbackupSshKeyTempFile.Name()))
		borgCommand.Stdout = os.Stdout
		borgCommand.Stdin = os.Stdin
		borgCommand.Stderr = os.Stderr
		err = borgCommand.Run()
		if err != nil {
			os.Remove(borgbackupSshKeyTempFile.Name())
			log.Fatal(err)
		}

		exec.Command("open", backupMountDir).Run()

		fmt.Printf("\n\nWhen you are finished, UNMOUNT the backup by running\n")
		fmt.Printf("     sku umount-backup %s\n", kubernetesNode)
	},
}

var umountCommand = &cobra.Command{
	Use:     "umount-backup",
	Aliases: []string{"umount-backup", "umount"},
	Short:   "ALPHA: (sandstorm) Unmount the backup at ~/src/k8s/backup/[nodename]",
	Long: `
ALPHA Quality!
`,
	Example: `
	sku umount-backup k3s2021-1
	sku umount-backup k3s2021-2
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		kubernetesNode := args[0]

		userHomeDir, _ := os.UserHomeDir()
		backupMountDir := userHomeDir + "/src/k8s/backup/" + kubernetesNode

		umountCommand := exec.Command("diskutil", "unmount", "force", backupMountDir)
		umountCommand.Stdout = os.Stdout
		umountCommand.Stdin = os.Stdin
		umountCommand.Stderr = os.Stderr
		err := umountCommand.Run()
		if err != nil {
			log.Fatal(err)
		}
	},
}
