package database

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/logrusorgru/aurora/v3"
	"github.com/phayes/freeport"
	"github.com/sandstorm/sku/pkg/kubernetes"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func MysqlDatabaseConnectionThroughPod(dbHost, dbName, dbUser, dbPassword string) (int, *sql.DB, *exec.Cmd, error) {
	return databaseConnectionThroughPod(dbHost, dbName, dbUser, dbPassword, 3306, func(localDbProxyPort int) (*sql.DB, error) {
		return sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(127.0.0.1:%d)/%s", dbUser, dbPassword, localDbProxyPort, dbName))
	})
}

func PostgresDatabaseConnectionThroughPod(dbHost, dbName, dbUser, dbPassword string) (int, *sql.DB, *exec.Cmd, error) {
	// see https://github.com/jackc/pgx/blob/master/stdlib/sql.go
	return databaseConnectionThroughPod(dbHost, dbName, dbUser, dbPassword, 5432, func(localDbProxyPort int) (*sql.DB, error) {
		return sql.Open("pgx", fmt.Sprintf("postgres://%s:%s@127.0.0.1:%d/%s", dbUser, dbPassword, localDbProxyPort, dbName))
	})
}

func databaseConnectionThroughPod(dbHost, dbName, dbUser, dbPassword string, dbPort int, sqlConnectionFactory func(localDbProxyPort int) (*sql.DB, error)) (int, *sql.DB, *exec.Cmd, error) {
	currentContext := kubernetes.KubernetesApiConfig().CurrentContext
	k8sContextDefinition := kubernetes.KubernetesApiConfig().Contexts[currentContext]

	fmt.Printf("1) K8S namespace %s in context %s\n", aurora.Green(k8sContextDefinition.Namespace), aurora.Green(currentContext))
	fmt.Println("")
	fmt.Println("   We will connect to the database by adding a Debug Container to an already-running Pod")
	fmt.Println("   in the namespace, so that we won't have problems with Network Policies etc.")
	fmt.Println("")
	fmt.Println()
	proxyPodName := kubernetes.SelectPod("Please select a Pod to use as a proxy for connecting to the Database")

	fmt.Println("2) Database connection parameters")
	fmt.Println("")

	fmt.Printf("   We will use the following database credentials to connect from within the Pod %s:\n", aurora.Green(proxyPodName))
	fmt.Println("")
	fmt.Println("")
	fmt.Printf("  - DB Host: %s\n", aurora.Green(dbHost))
	fmt.Printf("  - DB Name: %s\n", aurora.Green(dbName))
	fmt.Printf("  - DB User: %s\n", aurora.Green(dbUser))
	fmt.Printf("  - DB Password length: %s\n", aurora.Green(strconv.Itoa(len(dbPassword))))

	//=================================
	// Connect to DB via Proxy Pod
	//=================================
	fmt.Println("3) Trying to connect...")
	fmt.Println("")
	kubectlDebug := exec.Command(
		"kubectl",
		"debug",
		proxyPodName,
		"--image=alpine/socat",
		"--image-pull-policy=Always",
		"--arguments-only=true",
		"--",
		// here follow the socat arguments
		fmt.Sprintf("tcp-listen:%d,fork,reuseaddr", dbPort),
		fmt.Sprintf("tcp-connect:%s:%d", dbHost, dbPort),
	)
	err := kubectlDebug.Run()
	if err != nil {
		return 0, nil, nil, fmt.Errorf("%s could not run kubectl debug:\n    kubectl debug %v\n    %v\n", aurora.Red("ERROR:"), strings.Join(kubectlDebug.Args, " "), err)
	}
	fmt.Println("  - Started kubectl debug")

	localDbProxyPort, err := freeport.GetFreePort()
	if err != nil {
		return 0, nil, nil, fmt.Errorf("%s did not find a free port:\n    %v\n", aurora.Red("ERROR:"), err)
	}

	kubectlPortForward := exec.Command(
		"kubectl",
		"port-forward",
		fmt.Sprintf("pod/%s", proxyPodName),
		fmt.Sprintf("%d:%d", localDbProxyPort, dbPort),
	)
	kubectlPortForward.Stdout = os.Stdout
	kubectlPortForward.Stderr = os.Stderr

	err = kubectlPortForward.Start()
	if err != nil {
		return 0, nil, nil, fmt.Errorf("%s could not run kubectl port-forward:\n    kubectl port-forward %v\n    %v\n", aurora.Red("ERROR:"), strings.Join(kubectlPortForward.Args, " "), err)
	}
	fmt.Println("  - Started kubectl port-forward")

	db, err := sqlConnectionFactory(localDbProxyPort)
	if err != nil {
		kubectlPortForward.Process.Kill()
		return 0, nil, nil, fmt.Errorf("%s connection could not be created:\n    %v\n", aurora.Red("ERROR:"), err)
	}

	waitTime, _ := time.ParseDuration("1s")
	for {
		err = db.Ping()
		if err == nil {
			break
		}

		time.Sleep(waitTime)
		fmt.Println("- Waiting for Database to be available")
	}
	fmt.Println("- Got SQL connection")

	return localDbProxyPort, db, kubectlPortForward, nil
}
