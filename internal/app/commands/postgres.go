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
	"github.com/logrusorgru/aurora/v3"
	"github.com/sandstorm/sku/pkg/database"
	"github.com/sandstorm/sku/pkg/kubernetes"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
)

func BuildPostgresCommand() *cobra.Command {
	dbHost := ""
	dbName := ""
	dbUser := ""
	dbPassword := ""

	var postgresCommand = &cobra.Command{
		Use:                   "postgres [cli|pgcli|beekeeper] (extra-params)",
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"cli", "pgcli", "beekeeper"},
		Args:                  cobra.MinimumNArgs(1),
		Short:                 "Build a postgres connection and enter it via one of the given tools",
		Long: `
drop into a MySQL CLI to the given target
`,
		Run: func(cmd *cobra.Command, args []string) {
			allowedArgs := map[string]bool{
				"cli":       true,
				"pgcli":     true,
				"beekeeper": true,
			}
			if !allowedArgs[args[0]] {
				fmt.Printf("The tool %s is not supported. specify cli or mycli instead.\n", args[0])
				os.Exit(1)
			}

			dbHost = kubernetes.EvalScriptParameter(dbHost)
			dbName = kubernetes.EvalScriptParameter(dbName)
			dbUser = kubernetes.EvalScriptParameter(dbUser)
			dbPassword = kubernetes.EvalScriptParameter(dbPassword)

			localDbProxyPort, db, kubectlPortForward, err := database.DatabaseConnectionThroughPod(dbHost, dbName, dbUser, dbPassword, 5432)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			defer kubectlPortForward.Process.Kill()
			defer db.Close()

			switch args[0] {
			case "cli":
				psqlArgs := []string{
					"--host=127.0.0.1",
					fmt.Sprintf("--port=%d", localDbProxyPort),
					fmt.Sprintf("--username=%s", dbUser),
					dbName,
				}
				psqlArgs = append(psqlArgs, args[1:]...)

				psql := exec.Command(
					"psql",
					psqlArgs...,
				)
				psql.Env = append(os.Environ(),
					fmt.Sprintf("PGPASSWORD=%s", dbPassword),
				)
				psql.Stdout = os.Stdout
				psql.Stderr = os.Stderr
				psql.Stdin = os.Stdin

				psql.Run()

				break

			case "pgcli":
				pgcliArgs := []string{
					"--host", "127.0.0.1",
					"--port", strconv.Itoa(localDbProxyPort),
					"--user", dbUser,
					"--password", dbPassword,
					dbName,
				}
				pgcliArgs = append(pgcliArgs, args[1:]...)

				mycli := exec.Command(
					"pgcli",
					pgcliArgs...,
				)
				mycli.Stdout = os.Stdout
				mycli.Stderr = os.Stderr
				mycli.Stdin = os.Stdin

				mycli.Run()
				break

			case "beekeeper":
				openBeekeeper := exec.Command(
					"open",
					"/Applications/Beekeeper Studio.app",
				)
				openBeekeeper.Stdout = os.Stdout
				openBeekeeper.Stderr = os.Stderr
				openBeekeeper.Stdin = os.Stdin

				openBeekeeper.Run()

				fmt.Println(aurora.Bold("For Beekeeper Studio, you need to paste the following connection string:"))
				fmt.Println(aurora.Green(fmt.Sprintf("postgres://%s:%s@127.0.0.1:%d/%s", dbUser, dbPassword, localDbProxyPort, dbName)))
				fmt.Println(aurora.Bold("Keep this shell open as long as you want the DB connection to survive."))
				fmt.Println(aurora.Bold("Press Ctrl-C to close."))

				c := make(chan os.Signal)
				signal.Notify(c, os.Interrupt, syscall.SIGTERM)
				<-c

				break
			}
		},
	}

	postgresCommand.Flags().StringVarP(&dbHost, "dbHost", "", "eval:configmap('db').DB_HOST", "filename that contains the configuration to apply")
	postgresCommand.Flags().StringVarP(&dbName, "dbName", "", "eval:configmap('db').DB_NAME", "filename that contains the configuration to apply")
	postgresCommand.Flags().StringVarP(&dbUser, "dbUser", "", "eval:configmap('db').DB_USER", "filename that contains the configuration to apply")
	postgresCommand.Flags().StringVarP(&dbPassword, "dbPassword", "", "eval:secret('db').DB_PASSWORD", "filename that contains the configuration to apply")

	return postgresCommand
}

func init() {
	RootCmd.AddCommand(BuildPostgresCommand())
}