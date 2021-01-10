# Sandstorm Kubernetes Tools

**Convenience tools to interact with Kubernetes**

This project provides a command line tool called `sku` to interact with Kubernetes.

We provide convenience tooling to switch contexts and namespaces, enter containers, display logs, and many more.

## Installation

To install the `sku` binary tool on OSX, we recommend to use homebrew:

```bash
brew install sandstorm/tap/sku
```

**NEW: We suggest to set up [shell autocompletion](https://sandstorm.github.io/sku/#/autocompletion) after installing.**

Then, use `sku help` to get inline help, or e.g. run `sku ns` to list all namespaces in your current Kubernetes cluster.

## Features

The following commands are supported:

- [sku add-config](https://sandstorm.github.io/sku/#/context-and-ns?id=sku-add-config)
- [sku context](https://sandstorm.github.io/sku/#/context-and-ns?id=sku-context)
- [sku ns](https://sandstorm.github.io/sku/#/context-and-ns?id=sku-ns)
- [sku enter](https://sandstorm.github.io/sku/#/enter)
- [sku logs](https://sandstorm.github.io/sku/#/logs)
- [**NEW:** sku mysql](https://sandstorm.github.io/sku/#/database?id=entering-a-mysql-database)
- [**NEW:** sku postgres](https://sandstorm.github.io/sku/#/database?id=entering-a-postgres-database)
- [**WIP:** sku restore](https://sandstorm.github.io/sku/#/restore)

Additionally, some [alpha features](alpha.md) exist.

## Similar Projects

- [kubectx and kubens](https://github.com/ahmetb/kubectx/) to switch the Kubernetes context and namespace

## License

This project is licensed under the MIT license.