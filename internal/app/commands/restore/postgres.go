package restore

import (
	"database/sql"
	"fmt"
	"github.com/logrusorgru/aurora"
	"github.com/manifoldco/promptui"
	"github.com/sandstorm/sku/pkg/database"
	"github.com/sandstorm/sku/pkg/kubernetes"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func BuildPostgresCommand() *cobra.Command {
	dbHost := ""
	dbName := ""
	dbUser := ""
	dbPassword := ""
	restoreBackupPath := ""

	mariadbCommand := &cobra.Command{
		Use:   "postgres",
		Short: "Import a Postgres Dump",
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
				fmt.Println(aurora.Bold("Restoring Postgres to the Kubernetes cluster"))
				fmt.Println(aurora.Bold("============================================"))
				fmt.Println("This is an interactive wizard guiding you through restoring a Postgres database.")
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

				localDbProxyPort, db, kubectlPortForward, err := database.PostgresDatabaseConnectionThroughPod(dbHost, dbName, dbUser, dbPassword)
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

				pgDump := exec.Command(
					"pg_dump",
					"-h", "127.0.0.1",
					"-p", strconv.Itoa(localDbProxyPort),
					"-U", dbUser,
					"--format=plain",
					"--no-owner",
					"--no-privileges",
					"-f", fmt.Sprintf("%s/backup.sql", restoreBackupPath),
					dbName,
				)
				pgDump.Env = append(os.Environ(),
					fmt.Sprintf("PGPASSWORD=%s", dbPassword),
				)
				pgDump.Stdout = os.Stdout
				pgDump.Stderr = os.Stderr

				fmt.Println("- Starting to execute SQL backup")
				err = pgDump.Run()
				if err != nil {
					fmt.Printf("%s could not run mysqldump:\n    pg_dump %v\n    %v\n", aurora.Red("ERROR:"), strings.Join(pgDump.Args, " "), err)
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

				err = emptyPostgresDatabase(db, dbName)
				if err != nil {
					fmt.Printf("%s could not empty DB:\n    %v\n", aurora.Red("ERROR:"), err)
					return 1
				}

				//=================================
				// Import into database
				//=================================
				postgresImport := exec.Command(
					"bash",
					"-c",
					fmt.Sprintf(
						"psql -h 127.0.0.1 -p %d -U %s %s < %s",
						localDbProxyPort,
						dbUser,
						dbName,
						sqlFileName,
					),
				)
				postgresImport.Stdout = os.Stdout
				postgresImport.Stderr = os.Stderr
				postgresImport.Env = append(os.Environ(),
					fmt.Sprintf("PGPASSWORD=%s", dbPassword),
				)

				fmt.Println("- Importing SQL ")
				err = postgresImport.Run()
				if err != nil {
					fmt.Printf("%s could not import DB:\n    Command executed: %s\n    Error: %v\n", aurora.Red("ERROR:"), postgresImport.String(), err)
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

func emptyPostgresDatabase(db *sql.DB, dbName string) error {
	// Read all table names
	tables, err := db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = 'public';")
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

	for _, tableName := range tableNames {
		_, err = db.Exec(fmt.Sprintf("drop table if exists %s cascade;", tableName))
		if err != nil {
			return err
		}
	}

	return nil
}
