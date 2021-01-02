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

package mysql

import (
	"fmt"
	"github.com/sandstorm/sku/pkg/database"
	"github.com/sandstorm/sku/pkg/kubernetes"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
)

func BuildCliCommand() *cobra.Command {
	dbHost := ""
	dbName := ""
	dbUser := ""
	dbPassword := ""

	var cliCommand = &cobra.Command{
		Use:   "cli",
		Short: "drop into a MySQL CLI to the given target",
		Long: `
`,
		Run: func(cmd *cobra.Command, args []string) {
			dbHost = kubernetes.EvalScriptParameter(dbHost)
			dbName = kubernetes.EvalScriptParameter(dbName)
			dbUser = kubernetes.EvalScriptParameter(dbUser)
			dbPassword = kubernetes.EvalScriptParameter(dbPassword)

			localDbProxyPort, db, kubectlPortForward, err :=  database.DatabaseConnectionThroughPod(dbHost, dbName, dbUser, dbPassword)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			defer kubectlPortForward.Process.Kill()
			defer db.Close()

			mysqlArgs := []string{
				"--host=127.0.0.1",
				fmt.Sprintf("--port=%d", localDbProxyPort),
				fmt.Sprintf("--user=%s", dbUser),
				fmt.Sprintf("--password=%s", dbPassword),
				dbName,
			}
			mysqlArgs = append(mysqlArgs, args...)

			mysql := exec.Command(
				"mysql",
				mysqlArgs...
			)
			mysql.Stdout = os.Stdout
			mysql.Stderr = os.Stderr
			mysql.Stdin = os.Stdin

			mysql.Run()
		},
	}

	cliCommand.Flags().StringVarP(&dbHost, "dbHost", "", "eval:configmap('db').DB_HOST", "filename that contains the configuration to apply")
	cliCommand.Flags().StringVarP(&dbName, "dbName", "", "eval:configmap('db').DB_NAME", "filename that contains the configuration to apply")
	cliCommand.Flags().StringVarP(&dbUser, "dbUser", "", "eval:configmap('db').DB_USER", "filename that contains the configuration to apply")
	cliCommand.Flags().StringVarP(&dbPassword, "dbPassword", "", "eval:secret('db').DB_PASSWORD", "filename that contains the configuration to apply")


	return cliCommand
}
