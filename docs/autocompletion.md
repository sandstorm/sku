# Setting Up Autocompletion for SKU

We support autocompletion for *bash*, *zsh* and *fish* shells.

## Bash

**To load completions for each session, execute once:**

Linux:

```bash
sku completion bash > /etc/bash_completion.d/sku
```

MacOS:

```bash
sku completion bash > /usr/local/etc/bash_completion.d/sku
```

**You will need to start a new shell for this setup to take effect.**

## ZSH

If shell completion is not already enabled in your environment you will need
to enable it. You can execute the following once:

```zsh
echo "autoload -U compinit; compinit" >> ~/.zshrc
```

NOTE: If you use [oh-my-zsh](https://ohmyz.sh/), you **do not** need to do this, as this is already set up. 

**To load completions for each session, execute once:**

```zsh
sku completion zsh > "${fpath[1]}/_sku"
```

**You will need to start a new shell for this setup to take effect.**

## Fish

**To load completions for each session, execute once:**

```fish
sku completion fish > ~/.config/fish/completions/sku.fish
```

**You will need to start a new shell for this setup to take effect.**
