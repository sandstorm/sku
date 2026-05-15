package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/logrusorgru/aurora/v3"
	"github.com/manifoldco/promptui"
	"github.com/sandstorm/sku/pkg/kubernetes"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var natsUserGVR = schema.GroupVersionResource{
	Group:    "nats.k8s.sandstorm.de",
	Version:  "v1alpha1",
	Resource: "natsusers",
}

// Top-level nats CLI subcommands used for Layer 1 completion.
var natsTopLevelCommands = []string{
	"account", "audit", "auth", "bench",
	"consumer", "context", "counter", "errors",
	"events", "kv", "latency", "object",
	"publish", "reply", "request", "rtt",
	"schema", "server", "service", "stream",
	"subscribe", "top", "trace",
}

// Subcommands that need to be run from the NATS system account
// (e.g. they publish/subscribe under $SYS.>). For these we
// auto-prefer NatsUsers whose status.isSystemAccount is true.
var natsServerAdminCommands = map[string]bool{
	"server":  true,
	"account": true,
	"events":  true,
	"top":     true,
	"trace":   true,
	"audit":   true,
	"latency": true,
}

func BuildNatsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "nats [--server URL] [--user NAME] -- <nats cli args...>",
		Short:              "Connect to a NATS cluster as a NatsUser from the current namespace",
		DisableFlagParsing: true,
		Long: `
Run the 'nats' CLI against a NatsUser in the current Kubernetes namespace.

Discovers the NatsUser, extracts its nkey-seed into a 0600 tempfile that is
shredded on exit (including on Ctrl-C), reads the server URL from
NatsUser.status.connectionURLs, and execs 'nats' with your arguments.

Flags consumed by sku (everything else is forwarded to 'nats'):
  --server URL       override server URL
  --user NAME        NatsUser name (skip interactive selection)
`,
		Example: `
  sku nats sub ">"
  sku nats --user sandstorm-admin pub foo bar
  sku nats --server tls://your-server stream ls
`,
		ValidArgsFunction: natsValidArgs,
		Run: func(cmd *cobra.Command, args []string) {
			serverOverride, userOverride, rest := splitNatsArgs(args)
			runNats(serverOverride, userOverride, rest)
		},
	}
	return cmd
}

func init() {
	RootCmd.AddCommand(BuildNatsCommand())
}

// splitNatsArgs pulls out sku-specific flags (--server, --user) and returns
// the remaining args verbatim for the nats CLI. Supports both
// '--server X' and '--server=X' forms.
func splitNatsArgs(args []string) (server string, user string, rest []string) {
	rest = make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--server":
			if i+1 < len(args) {
				server = args[i+1]
				i++
			}
		case strings.HasPrefix(a, "--server="):
			server = strings.TrimPrefix(a, "--server=")
		case a == "--user":
			if i+1 < len(args) {
				user = args[i+1]
				i++
			}
		case strings.HasPrefix(a, "--user="):
			user = strings.TrimPrefix(a, "--user=")
		default:
			rest = append(rest, a)
		}
	}
	return
}

func natsValidArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Completing a flag value?
	if len(args) > 0 {
		switch args[len(args)-1] {
		case "--user":
			return listNatsUserNames(), cobra.ShellCompDirectiveNoFileComp
		case "--server":
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	}
	// Suggest flags when current word starts with '-'.
	if strings.HasPrefix(toComplete, "-") {
		return []string{"--server", "--user"}, cobra.ShellCompDirectiveNoFileComp
	}
	// Strip sku-specific tokens to figure out how many "real" args precede.
	_, _, rest := splitNatsArgs(args)
	if len(rest) == 0 {
		return natsTopLevelCommands, cobra.ShellCompDirectiveNoFileComp
	}
	return nil, cobra.ShellCompDirectiveDefault
}

func listNatsUserNames() []string {
	ns := kubernetes.CurrentNamespace()
	if ns == "" {
		return nil
	}
	dyn, err := dynamic.NewForConfig(kubernetes.KubernetesRestConfig())
	if err != nil {
		return nil
	}
	list, err := dyn.Resource(natsUserGVR).Namespace(ns).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(list.Items))
	for _, u := range list.Items {
		names = append(names, u.GetName())
	}
	return names
}

func runNats(serverOverride, userOverride string, args []string) {
	ns := kubernetes.CurrentNamespace()
	if ns == "" {
		fail("could not determine current namespace from kubeconfig")
	}

	dyn, err := dynamic.NewForConfig(kubernetes.KubernetesRestConfig())
	if err != nil {
		fail("could not build dynamic client: %v", err)
	}

	user, err := selectNatsUser(dyn, ns, userOverride, isServerAdminCommand(args))
	if err != nil {
		fail("%v", err)
	}

	secretName, secretKey := nkeySecretRef(user)
	secret, err := kubernetes.KubernetesClientset().CoreV1().Secrets(ns).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		fail("could not fetch nkey secret %s/%s: %v", ns, secretName, err)
	}
	seed, ok := secret.Data[secretKey]
	if !ok || len(seed) == 0 {
		fail("secret %s/%s has no key %q (or it is empty)", ns, secretName, secretKey)
	}

	serverURL := serverOverride
	if serverURL == "" {
		serverURL, err = resolveServerURL(user)
		if err != nil {
			fail("%v", err)
		}
	}

	// Write the seed to a short-lived 0600 tempfile, then unlink it
	// shortly after the child starts (natscli reads the seed once at
	// startup, then doesn't need the path anymore). The window the file
	// exists on disk is bounded by 'nats' startup time.
	tmp, err := os.CreateTemp("", "sku-nats-nkey-*")
	if err != nil {
		fail("could not create nkey tempfile: %v", err)
	}
	seedPath := tmp.Name()
	if err := os.Chmod(seedPath, 0600); err != nil {
		tmp.Close()
		os.Remove(seedPath)
		fail("could not chmod nkey tempfile: %v", err)
	}
	if _, err := tmp.Write(seed); err != nil {
		tmp.Close()
		os.Remove(seedPath)
		fail("could not write nkey tempfile: %v", err)
	}
	tmp.Close()

	var removeOnce sync.Once
	removeSeed := func() { removeOnce.Do(func() { _ = os.Remove(seedPath) }) }
	defer removeSeed()

	natsArgs := append([]string{"--nkey", seedPath, "--server", serverURL}, args...)
	natsCmd := exec.Command("nats", natsArgs...)
	natsCmd.Stdin = os.Stdin
	natsCmd.Stdout = os.Stdout
	natsCmd.Stderr = os.Stderr
	// Prevent any locally-configured 'nats context' from leaking creds.
	natsCmd.Env = append(os.Environ(), "NATS_CONTEXT=")

	fmt.Fprintf(os.Stderr, "%s NatsUser %s @ %s\n",
		aurora.Yellow("INFO:"), aurora.Green(user.name), aurora.Green(serverURL))

	if err := natsCmd.Start(); err != nil {
		fail("could not start 'nats' (is it installed and on PATH?): %v", err)
	}

	// natscli reads the seed once at startup. Give it a moment, then
	// unlink — even on Ctrl-C the deferred removeSeed runs.
	go func() {
		time.Sleep(1 * time.Second)
		removeSeed()
	}()

	// Forward SIGINT/SIGTERM to the child, then clean up.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		s := <-sigCh
		removeSeed()
		if natsCmd.Process != nil {
			_ = natsCmd.Process.Signal(s)
		}
	}()

	if err := natsCmd.Wait(); err != nil {
		removeSeed()
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fail("nats CLI failed: %v", err)
	}
}

type natsUserRef struct {
	name      string
	namespace string
	// raw unstructured object — we read status/spec dynamically.
	obj map[string]interface{}
}

func selectNatsUser(dyn dynamic.Interface, ns, override string, preferSystemAccount bool) (*natsUserRef, error) {
	if override != "" {
		u, err := dyn.Resource(natsUserGVR).Namespace(ns).Get(context.Background(), override, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("NatsUser %s/%s not found: %v", ns, override, err)
		}
		return &natsUserRef{name: u.GetName(), namespace: ns, obj: u.Object}, nil
	}

	list, err := dyn.Resource(natsUserGVR).Namespace(ns).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not list NatsUsers in %s: %v", ns, err)
	}
	if len(list.Items) == 0 {
		return nil, fmt.Errorf("no NatsUser found in namespace %s", ns)
	}

	items := list.Items
	if preferSystemAccount {
		sys := make([]unstructured.Unstructured, 0, len(items))
		for _, u := range items {
			if isSystemAccountUser(u.Object) {
				sys = append(sys, u)
			}
		}
		if len(sys) == 0 {
			return nil, fmt.Errorf("this nats subcommand needs a system-account NatsUser (status.isSystemAccount=true), but none was found in namespace %s", ns)
		}
		fmt.Fprintf(os.Stderr, "%s server-admin command — restricting to system-account NatsUsers (status.isSystemAccount=true)\n",
			aurora.Yellow("INFO:"))
		items = sys
	}

	if len(items) == 1 {
		u := items[0]
		fmt.Fprintf(os.Stderr, "%s using NatsUser in %s: %s\n",
			aurora.Yellow("INFO:"), ns, u.GetName())
		return &natsUserRef{name: u.GetName(), namespace: ns, obj: u.Object}, nil
	}

	names := make([]string, 0, len(items))
	for _, u := range items {
		names = append(names, u.GetName())
	}
	prompt := promptui.Select{
		Label: aurora.Bold("Select NatsUser"),
		Items: names,
	}
	idx, _, err := prompt.Run()
	if err != nil {
		return nil, fmt.Errorf("selection cancelled: %v", err)
	}
	u := items[idx]
	return &natsUserRef{name: u.GetName(), namespace: ns, obj: u.Object}, nil
}

func isSystemAccountUser(obj map[string]interface{}) bool {
	status, ok := obj["status"].(map[string]interface{})
	if !ok {
		return false
	}
	v, _ := status["isSystemAccount"].(bool)
	return v
}

// isServerAdminCommand inspects the args destined for the nats CLI and
// returns true when the first non-flag token is a subcommand that
// requires the NATS system account.
func isServerAdminCommand(args []string) bool {
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		return natsServerAdminCommands[a]
	}
	return false
}

// nkeySecretRef returns the secret name + key the operator creates for a
// NatsUser: secret "<user>-nats-nkey", key "nkey-seed".
func nkeySecretRef(u *natsUserRef) (string, string) {
	return u.name + "-nats-nkey", "nkey-seed"
}

func resolveServerURL(u *natsUserRef) (string, error) {
	status, ok := u.obj["status"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("NatsUser %s has no status yet; pass --server", u.name)
	}
	rawURLs, ok := status["connectionURLs"].([]interface{})
	if !ok || len(rawURLs) == 0 {
		return "", fmt.Errorf("NatsUser %s has no status.connectionURLs; pass --server", u.name)
	}
	urls := make([]string, 0, len(rawURLs))
	for _, v := range rawURLs {
		if s, ok := v.(string); ok {
			urls = append(urls, s)
		}
	}
	// Prefer secure-before-insecure, native-before-websocket.
	for _, scheme := range []string{"tls://", "wss://", "nats://", "ws://"} {
		for _, url := range urls {
			if strings.HasPrefix(url, scheme) {
				return url, nil
			}
		}
	}
	// No preferred scheme matched — prompt.
	prompt := promptui.Select{
		Label: aurora.Bold("Select connection URL"),
		Items: urls,
	}
	_, choice, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("no URL selected: %v", err)
	}
	return choice, nil
}

func fail(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "%s %s\n", aurora.Red("ERROR:"), fmt.Sprintf(format, args...))
	os.Exit(1)
}
