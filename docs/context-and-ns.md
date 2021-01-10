# kubectl contexts and namespaces

`kubectl` supports multiple servers and/or users by a feature called **contexts**. When working with contexts
(and namespaces) leads to long names.

You can use `sku add-config` to build up your multi-context `.kube/config` file; and then use `sku context` for showing
and switching contexts.

`sku ns` can be used for displaying and switching namespaces.

## sku add-config

This command allows adding an external kubeconfig file to the default `~/.kube/config`;
so you can actually merge multiple Kubernetes configs together which you got from different sources.

*Example*:

```bash
sku add-config path-to-additional-kubeconfig-file
```

## sku context

*This command allows to switch between different configured Kubernetes clusters*.

**List all configured contexts**: `sku context`

**Switch the active context**: `sku context [context-name]`

## sku ns

**List all namespaces**: `sku ns`

**Switch the active namespace**: `sku ns [namespace-name]`

