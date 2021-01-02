package restore

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/logrusorgru/aurora"
	"github.com/manifoldco/promptui"
	"github.com/sandstorm/sku/pkg/database"
	"github.com/sandstorm/sku/pkg/kubernetes"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"path"
	"path/filepath"
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

				dbHost = kubernetes.EvalScriptParameter(dbHost)
				dbName = kubernetes.EvalScriptParameter(dbName)
				dbUser = kubernetes.EvalScriptParameter(dbUser)
				dbPassword = kubernetes.EvalScriptParameter(dbPassword)

				localDbProxyPort, db, kubectlPortForward, err :=  database.DatabaseConnectionThroughPod(dbHost, dbName, dbUser, dbPassword)
				if err != nil {
					fmt.Println(err)
					return 1
				}
				defer kubectlPortForward.Process.Kill()
				defer db.Close()

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


