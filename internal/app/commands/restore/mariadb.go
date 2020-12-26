package restore

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/dop251/goja"
	_ "github.com/go-sql-driver/mysql"
	"github.com/logrusorgru/aurora"
	"github.com/manifoldco/promptui"
	"github.com/phayes/freeport"
	"github.com/sandstorm/sku/pkg/kubernetes"
	"github.com/spf13/cobra"
	clientV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func BuildMariadbCommand() *cobra.Command {
	dbHost := ""
	dbName := ""
	dbUser := ""
	dbPassword := ""
	restoreBackupPath := ""

	mariadbCommand := &cobra.Command{
		Use:   "mariadb",
		Short: "Import a Mariadb",
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
				fmt.Println(aurora.Bold("Restoring MariaDB to the Kubernetes cluster"))
				fmt.Println(aurora.Bold("==========================================="))
				fmt.Println("This is an interactive wizard guiding you through restoring a MySQL/MariaDB database.")
				fmt.Println("Before any destructive operation, you'll be asked whether you want to continue.")
				fmt.Println("")
				fmt.Println("")

				sqlFileName := args[0]
				if len(sqlFileName) == 0 {
					fmt.Printf("%s the SQL file must be given as parameter\n", aurora.Red("ERROR:"))
					return 1
				}

				fileStats, err := os.Stat(sqlFileName)
				if err != nil {
					fmt.Printf("%s SQL File %s not found:\n    %s\n", aurora.Red("ERROR:"), aurora.Bold(sqlFileName), err)
					return 1
				}
				if fileStats.IsDir() {
					fmt.Printf("%s SQL File %s is a directory, but needs to be a file\n", aurora.Red("ERROR:"), aurora.Bold(sqlFileName))
					return 1
				}

				currentContext := kubernetes.KubernetesApiConfig().CurrentContext
				k8sContextDefinition := kubernetes.KubernetesApiConfig().Contexts[currentContext]

				fmt.Printf("1) K8S namespace %s in context %s\n", aurora.Green(k8sContextDefinition.Namespace), aurora.Green(currentContext))
				fmt.Println("")
				fmt.Println("   We will connect to the database by adding a Debug Container to an already-running Pod")
				fmt.Println("   in the namespace, so that we won't have problems with Network Policies etc.")
				fmt.Println("")
				fmt.Println()
				podName := selectPod("Please select a Pod to use as a proxy for connecting to the Database")

				fmt.Println("2) Database connection parameters")
				fmt.Println("")
				fmt.Printf("   We will use the following database credentials to connect from within the Pod %s:\n", aurora.Green(podName))
				fmt.Println("")
				fmt.Println("")

				dbHost = execScriptParameter(dbHost)
				dbName = execScriptParameter(dbName)
				dbUser = execScriptParameter(dbUser)
				dbPassword = execScriptParameter(dbPassword)
				fmt.Printf("  - DB Host: %s\n", aurora.Green(dbHost))
				fmt.Printf("  - DB Name: %s\n", aurora.Green(dbName))
				fmt.Printf("  - DB User: %s\n", aurora.Green(dbUser))
				fmt.Printf("  - DB Password length: %s\n", aurora.Green(strconv.Itoa(len(dbPassword))))

				//=================================
				// Connect to DB via Proxy Pod
				//=================================
				fmt.Println("3) Trying to connect and creating a backup")
				fmt.Println("")
				kubectlDebug := exec.Command(
					"kubectl",
					"debug",
					podName,
					"--image=alpine/socat",
					"--image-pull-policy=Always",
					"--arguments-only=true",
					"--",
					// here follow the socat arguments
					"tcp-listen:3306,fork,reuseaddr",
					fmt.Sprintf("tcp-connect:%s:3306", dbHost),
				)
				err = kubectlDebug.Run()
				if err != nil {
					fmt.Printf("%s could not run kubectl debug:\n    kubectl debug %v\n    %v\n", aurora.Red("ERROR:"), strings.Join(kubectlDebug.Args, " "), err)
					return 1
				}
				fmt.Println("  - Started kubectl debug")

				localDbProxyPort, err := freeport.GetFreePort()
				if err != nil {
					fmt.Printf("%s did not find a free port:\n    %v\n", aurora.Red("ERROR:"), err)
					return 1
				}

				kubectlPortForward := exec.Command(
					"kubectl",
					"port-forward",
					fmt.Sprintf("pod/%s", podName),
					fmt.Sprintf("%d:3306", localDbProxyPort),
				)
				kubectlPortForward.Stdout = os.Stdout
				kubectlPortForward.Stderr = os.Stderr

				err = kubectlPortForward.Start()
				if err != nil {
					fmt.Printf("%s could not run kubectl port-forward:\n    kubectl port-forward %v\n    %v\n", aurora.Red("ERROR:"), strings.Join(kubectlPortForward.Args, " "), err)
					return 1
				}
				defer kubectlPortForward.Process.Kill()
				fmt.Println("  - Started kubectl port-forward")

				db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(127.0.0.1:%d)/%s", dbUser, dbPassword, localDbProxyPort, dbName))
				if err != nil {
					fmt.Printf("%s mysql connection could not be created:\n    %v\n", aurora.Red("ERROR:"), err)
					return 1
				}
				defer db.Close()

				waitTime, _ := time.ParseDuration("1s")
				for {
					err = db.Ping()
					if err == nil {
						break
					}

					time.Sleep(waitTime)
					fmt.Println("- Waiting for MySQL to be available")
				}
				fmt.Println("- Got SQL connection")

				//=================================
				// Create SQL Backup
				//=================================
				fmt.Println("")
				fmt.Println("")
				fmt.Printf("4) %s and inport the SQL dump", aurora.Bold("Clear all data"))
				fmt.Println("")
				fmt.Println("   After doing an SQL dump, the database will be cleared, and the given data from the backup will be imported.")
				fmt.Println("")

				restoreBackupPath = path.Join(restoreBackupPath, time.Now().Format("01-02-2006-15-04-05")+"__"+k8sContextDefinition.Namespace)
				if err = os.MkdirAll(restoreBackupPath, os.ModePerm); err != nil {
					fmt.Printf("%s could not create %s:\n    %v\n", aurora.Red("ERROR:"), restoreBackupPath, err)
				}

				mysqlDump := exec.Command(
					"mysqldump",
					"--host=127.0.0.1",
					fmt.Sprintf("--port=%d", localDbProxyPort),
					fmt.Sprintf("--user=%s", dbUser),
					fmt.Sprintf("--password=%s", dbPassword),
					fmt.Sprintf("--result-file=%s/backup.sql", restoreBackupPath),
					dbName,
				)
				mysqlDump.Stdout = os.Stdout
				mysqlDump.Stderr = os.Stderr

				fmt.Println("- Starting to execute SQL backup")
				err = mysqlDump.Run()
				if err != nil {
					fmt.Printf("%s could not run mysqldump:\n    mysqldump %v\n    %v\n", aurora.Red("ERROR:"), strings.Join(mysqlDump.Args, " "), err)
					return 1
				}
				fmt.Println("- Finished to execute SQL backup")

				//=================================
				// Empty database
				//=================================
				prompt := promptui.Prompt{
					Label:     aurora.Bold("CLEAR THE DATABASE and IMPORT from backup?"),
					IsConfirm: true,
				}
				_, err = prompt.Run()
				if err != nil {
					fmt.Printf("user aborted.\n")
					return 1
				}

				err = emptyDatabase(db, dbName)
				if err != nil {
					fmt.Printf("%s could not empty DB:\n    %v\n", aurora.Red("ERROR:"), err)
					return 1
				}

				//=================================
				// Import into database
				//=================================
				mysqlImport := exec.Command(
					"bash",
					"-c",
					fmt.Sprintf(
						"mysql --host=127.0.0.1 --port=%d --user=%s --password=%s %s < %s",
						localDbProxyPort,
						dbUser,
						dbPassword,
						dbName,
						sqlFileName,
					),
				)
				mysqlImport.Stdout = os.Stdout
				mysqlImport.Stderr = os.Stderr

				fmt.Println("- Importing SQL ")
				err = mysqlImport.Run()
				if err != nil {
					fmt.Printf("%s could not import DB:\n    Command executed: %s\n    Error: %v\n", aurora.Red("ERROR:"), mysqlImport.String(), err)
					return 1
				}

				return 0
			})()
			os.Exit(resultCode)
		},
	}

	mariadbCommand.Flags().StringVarP(&dbHost, "dbHost", "", "eval:configmap('db').DB_HOST", "filename that contains the configuration to apply")
	mariadbCommand.Flags().StringVarP(&dbName, "dbName", "", "eval:configmap('db').DB_NAME", "filename that contains the configuration to apply")
	mariadbCommand.Flags().StringVarP(&dbUser, "dbUser", "", "eval:configmap('db').DB_USER", "filename that contains the configuration to apply")
	mariadbCommand.Flags().StringVarP(&dbPassword, "dbPassword", "", "eval:secret('db').DB_PASSWORD", "filename that contains the configuration to apply")
	userHomeDir, _ := os.UserHomeDir()
	mariadbCommand.Flags().StringVarP(&restoreBackupPath, "restoreBackupPath", "", filepath.Join(userHomeDir, "src/k8s/restore-backups"), "filename that contains the configuration to apply")

	return mariadbCommand
}

func emptyDatabase(db *sql.DB, dbName string) error {
	// Read all table names
	tables, err := db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = ?;", dbName)
	if err != nil {
		return err
	}
	tableNames := make([]string, 0, 10)
	defer tables.Close()
	for tables.Next() {
		var tableName string
		err := tables.Scan(&tableName)
		if err != nil {
			return err
		}
		tableNames = append(tableNames, tableName)
	}
	err = tables.Err()
	if err != nil {
		return err
	}

	// Drop all tables
	_, err = db.Exec("SET FOREIGN_KEY_CHECKS = 0;")
	if err != nil {
		return err
	}

	for _, tableName := range tableNames {
		_, err = db.Exec(fmt.Sprintf("drop table if exists %s;", tableName))
		if err != nil {
			return err
		}
	}

	_, err = db.Exec("SET FOREIGN_KEY_CHECKS = 1;")
	if err != nil {
		return err
	}

	return nil
}

func selectPod(promptLabel string) string {
	currentContext := kubernetes.KubernetesApiConfig().CurrentContext
	k8sContextDefinition := kubernetes.KubernetesApiConfig().Contexts[currentContext]

	// query for running pods in current namespace
	podList, _ := kubernetes.KubernetesClientset().CoreV1().Pods(k8sContextDefinition.Namespace).List(context.Background(), metav1.ListOptions{})

	podNames := make([]string, 0, len(podList.Items))
	for _, pod := range podList.Items {
		if pod.Status.Phase == clientV1.PodRunning {
			podNames = append(podNames, pod.Name)
		}
	}

	prompt := promptui.Select{
		Label: aurora.Bold(promptLabel),
		Items: podNames,
	}

	_, result, err := prompt.Run()

	if err != nil {
		fmt.Printf("%s prompt failed:\n    %v\n", aurora.Red("ERROR:"), err)
		// TODO: get rid of os.Exit here (breaks the outer goroutines)
		os.Exit(1)
	}

	return result
}

func execScriptParameter(parameter string) string {
	if strings.HasPrefix(parameter, "eval:") {
		vm := goja.New()
		vm.Set("secret", func(secretName string) map[string]string {
			currentContext := kubernetes.KubernetesApiConfig().CurrentContext
			k8sContextDefinition := kubernetes.KubernetesApiConfig().Contexts[currentContext]
			secret, err := kubernetes.KubernetesClientset().CoreV1().Secrets(k8sContextDefinition.Namespace).Get(context.Background(), secretName, metav1.GetOptions{})
			if err != nil {
				fmt.Printf("%s Secret %s could not be fetched (evaluating %s):\n    %s\n", aurora.Red("ERROR:"), aurora.Bold(secretName), aurora.Bold(parameter), err)
				// TODO: get rid of os.Exit here (breaks the outer goroutines)
				os.Exit(1)
			}

			converted := make(map[string]string, len(secret.Data))
			for k, v := range secret.Data {
				converted[k] = string(v)
			}
			return converted
		})

		vm.Set("configmap", func(configmapName string) map[string]string {
			currentContext := kubernetes.KubernetesApiConfig().CurrentContext
			k8sContextDefinition := kubernetes.KubernetesApiConfig().Contexts[currentContext]
			configMap, err := kubernetes.KubernetesClientset().CoreV1().ConfigMaps(k8sContextDefinition.Namespace).Get(context.Background(), configmapName, metav1.GetOptions{})
			if err != nil {
				fmt.Printf("%s Config Map %s could not be fetched (evaluating %s):\n    %s\n", aurora.Red("ERROR:"), aurora.Bold(configmapName), aurora.Bold(parameter), err)
				// TODO: get rid of os.Exit here (breaks the outer goroutines)
				os.Exit(1)
			}

			return configMap.Data
		})

		v, err := vm.RunString(parameter[5:])
		if err != nil {
			fmt.Printf("%s Could not evaluate %s:\n    %s\n", aurora.Red("ERROR:"), aurora.Bold(parameter), err)
			// TODO: get rid of os.Exit here (breaks the outer goroutines)
			os.Exit(1)
		}

		converted, isOk := v.Export().(string)
		if !isOk {
			fmt.Printf("%s Could not convert %s:\n    Value: %+v\n", aurora.Red("ERROR:"), aurora.Bold(parameter), v)
			// TODO: get rid of os.Exit here (breaks the outer goroutines)
			os.Exit(1)
		}
		return converted
	} else {
		return parameter
	}

}
