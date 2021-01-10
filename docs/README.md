# Sandstorm Kubernetes Tools

**Convenience tools to interact with Kubernetes**

This project provides a command line tool called `sku` to interact with Kubernetes.

We provide convenience tooling to switch contexts and namespaces, enter containers, display logs, and many more.

## Installation

To install the `sku` binary tool on OSX, we recommend to use homebrew:

```bash
brew install sandstorm/tap/sku
```

**NEW: We suggest to set up [shell autocompletion](autocompletion.md) after installing.**

Then, use `sku help` to get inline help, or e.g. run `sku ns` to list all namespaces in your current Kubernetes cluster.

## Features

The following commands are supported:

- [sku add-config](context-and-ns.md#sku-add-config)
- [sku context](context-and-ns.md#sku-context)
- [sku ns](context-and-ns.md#sku-ns)
- [sku enter](enter.md)
- [sku logs](logs.md)
- [**NEW:** sku mysql](database.md#entering-a-mysql-database)
- [**NEW:** sku postgres](database.md#entering-a-postgres-database)
- [**NEW:** sku restore](restore.md)

Additionally, some [alpha features](alpha.md) exist.

## Similar Projects

- [kubectx and kubens](https://github.com/ahmetb/kubectx/) to switch the Kubernetes context and namespace

## License

This project is licensed under the MIT license.