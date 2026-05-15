# Setting Up Autocompletion for SKU

> **zsh + [oh-my-zsh](https://ohmyz.sh/) only.** That's our supported setup.

## Install / refresh

Paste this into your shell. It removes any stale `_sku` from anywhere on
`$fpath` and installs a fresh one into oh-my-zsh's custom completions
directory (already on `$fpath`, so no `.zshrc` edits needed):

```zsh
# uninstall previously installed _sku autocompleter if any
for d in $fpath; do [[ -f "$d/_sku" ]] && rm -f "$d/_sku"; done
# install completion
mkdir -p "$ZSH_CUSTOM/completions"
sku completion zsh > "$ZSH_CUSTOM/completions/_sku"
rm -f ~/.zcompdump* && exec zsh
```

`$ZSH_CUSTOM` defaults to `~/.oh-my-zsh/custom`, so the script lands at
`~/.oh-my-zsh/custom/completions/_sku`.

## Why this location?

oh-my-zsh prepends `$ZSH_CUSTOM/completions` to `$fpath` and calls
`compinit` for you — it's the idiomatic home for third-party zsh
completions when oh-my-zsh is in use. No plugin to maintain, no `.zshrc`
edits, survives oh-my-zsh upgrades.

Avoid:

- `${fpath[1]}/_sku` — whatever happens to be first on `$fpath`, often a
  shared Homebrew or system dir that gets overwritten on update.
- `~/.oh-my-zsh/custom/plugins/<name>/_sku` — only works if that plugin
  is enabled in `plugins=(...)`.

## Verifying

```zsh
whence -v _sku    # → /Users/you/.oh-my-zsh/custom/completions/_sku
sku <TAB>         # lists subcommands
sku nats <TAB>    # passthrough to the nats CLI (Layer 2)
```

If `whence -v _sku` reports a different path, the cleanup loop missed a
directory — delete that file manually and re-run.
