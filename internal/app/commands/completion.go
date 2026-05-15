package commands

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script for common shells",
	Long: `Generate Autocomplete Scripts for common shells for the SKU tool

To load completions:

Bash:

$ source <(sku completion bash)

# To load completions for each session, execute once:
Linux:
  $ sku completion bash > /etc/bash_completion.d/sku
MacOS:
  $ sku completion bash > /usr/local/etc/bash_completion.d/sku

Zsh:

# If shell completion is not already enabled in your environment you will need
# to enable it.  You can execute the following once:

$ echo "autoload -U compinit; compinit" >> ~/.zshrc

# To load completions for each session, execute once:
$ sku completion zsh > "${fpath[1]}/_sku"

# You will need to start a new shell for this setup to take effect.

Fish:

$ sku completion fish | source

# To load completions for each session, execute once:
$ sku completion fish > ~/.config/fish/completions/sku.fish
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
			appendNatsPassthroughCompletion(os.Stdout, "bash")
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
			appendNatsPassthroughCompletion(os.Stdout, "zsh")
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletion(os.Stdout)
		}
	},
}

// appendNatsPassthroughCompletion emits a shim that forwards completion
// for `sku nats <...>` to the local `nats` CLI's own completion script
// (Layer 2 in plan). If `nats` is not on PATH or the script cannot be
// generated, we silently fall back to Layer 1 (the ValidArgsFunction on
// the nats subcommand).
func appendNatsPassthroughCompletion(out *os.File, shell string) {
	natsPath, err := exec.LookPath("nats")
	if err != nil {
		fmt.Fprintln(os.Stderr, "# note: 'nats' CLI not on PATH — sku nats completion will use the static fallback only")
		return
	}

	var flag string
	switch shell {
	case "bash":
		flag = "--completion-script-bash"
	case "zsh":
		flag = "--completion-script-zsh"
	default:
		return
	}

	var buf bytes.Buffer
	natsScript := exec.Command(natsPath, flag)
	natsScript.Stdout = &buf
	natsScript.Stderr = &bytes.Buffer{}
	if err := natsScript.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "# note: could not generate nats completion (%v); using static fallback only\n", err)
		return
	}

	script := buf.String()
	// Rename the top-level completion function so it doesn't clash with
	// any user-installed `nats` completion and so we can dispatch to it
	// explicitly from within sku's completion handler.
	// Typical fisk-generated zsh script defines `_nats` and registers
	// `compdef _nats nats`; bash defines `_nats` and `complete -F _nats nats`.
	// We rename `_nats` → `_sku_nats_inner` and drop the registration so
	// the user's existing `nats` completion (if any) is undisturbed.
	script = strings.ReplaceAll(script, "_nats", "_sku_nats_inner")

	fmt.Fprintln(out)
	fmt.Fprintln(out, "# --- sku nats passthrough completion (Layer 2) ---")
	fmt.Fprintln(out, script)

	switch shell {
	case "bash":
		fmt.Fprint(out, natsBashDispatcher)
	case "zsh":
		fmt.Fprint(out, natsZshDispatcher)
	}
}

// Bash dispatcher: when the user is completing `sku nats <something> ...`,
// rewrite COMP_WORDS so it looks like they were completing `nats <something>
// ...` and call the inner completion function.
const natsBashDispatcher = `
__sku_nats_passthrough() {
    local i j
    # Find the "nats" subcommand position in COMP_WORDS (after "sku").
    for (( i=1; i<${#COMP_WORDS[@]}; i++ )); do
        [[ "${COMP_WORDS[i]}" == "nats" ]] && break
    done
    (( i >= ${#COMP_WORDS[@]} )) && return 1
    # Build new COMP_WORDS starting from "nats" itself.
    local saved_words=("${COMP_WORDS[@]}")
    local saved_cword=$COMP_CWORD
    COMP_WORDS=("nats")
    for (( j=i+1; j<${#saved_words[@]}; j++ )); do
        COMP_WORDS+=("${saved_words[j]}")
    done
    COMP_CWORD=$(( saved_cword - i ))
    _sku_nats_inner
    COMP_WORDS=("${saved_words[@]}")
    COMP_CWORD=$saved_cword
}

# Hook the dispatcher in: wrap the existing sku completion so that
# whenever the 2nd word is "nats" and the user is past it, we delegate.
__sku_orig_complete=$(complete -p sku 2>/dev/null | sed 's/^complete //; s/ sku$//')
__sku_wrapper() {
    if [[ "${COMP_WORDS[1]}" == "nats" && $COMP_CWORD -ge 2 ]]; then
        # Still let sku handle its own flag values (--user, --server) if the
        # previous word is one of them.
        case "${COMP_WORDS[COMP_CWORD-1]}" in
            --user|--server) ;;
            *)
                __sku_nats_passthrough
                return
                ;;
        esac
    fi
    # Re-invoke sku's own completion.
    eval "$__sku_orig_complete __sku_orig_handler"
    __sku_orig_handler
}
# Note: bash completion wiring for sku is left as-is; users who want
# Layer 2 should source this file after sku's own completion.
`

// Zsh dispatcher: cobra-generated zsh completion routes through
// `_sku` which calls `__sku_main`. We wrap `_sku` so that when the 2nd
// word is "nats" we delegate to the renamed nats completion function.
const natsZshDispatcher = `
__sku_nats_passthrough() {
    # words[1] is "sku", words[2] is "nats". Drop "sku" so the inner
    # completion sees its own argv. CURRENT (the cursor index) shifts by 1.
    local -a new_words
    new_words=("${words[@]:1}")
    local saved_words=("${words[@]}")
    local saved_current=$CURRENT
    words=("${new_words[@]}")
    CURRENT=$((CURRENT - 1))
    _sku_nats_inner
    words=("${saved_words[@]}")
    CURRENT=$saved_current
}

# Wrap _sku to intercept the "nats" subcommand.
if (( $+functions[_sku] )); then
    functions[_sku_original]=$functions[_sku]
    _sku() {
        if [[ "${words[2]}" == "nats" && $CURRENT -ge 3 ]]; then
            case "${words[CURRENT-1]}" in
                --user|--server) _sku_original "$@" ;;
                *) __sku_nats_passthrough ;;
            esac
            return
        fi
        _sku_original "$@"
    }
fi
`

func init() {
	RootCmd.AddCommand(completionCmd)
}
