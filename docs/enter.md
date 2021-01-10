# Entering a Kubernetes Pod

Enter an interactive shell in a pod of **the current namespace**.
To select the pods you want to enter, you'll see a choice list if
there is more than one pod accessible.

```bash
sku enter
```

Optionally, you can restrict the pod list by specifying a label
selector:

```bash
sku enter app=foo
sku enter app=foo,component=app
```

If `bash` is available, this is used; otherwise, `sh` is used.