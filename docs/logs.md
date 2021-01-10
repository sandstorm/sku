# Displaying Logs for a Kubernetes Pod

Show the logs of a pod of the current namespace.
To select the pods you want to get the logs for, you'll see a choice list:

```bash
sku logs
```

Optionally, you can restrict the pod list by specifying a label
selector:

```bash
sku logs app=foo
sku logs app=foo,component=app
```

