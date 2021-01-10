# Restoring backups

> **NOTE**: This section is **extremely opinionated** right now, fitting to the Sandstorm
> conventions. We'd love to refactor this to be more useful generally-purpose; please let us
> know what you need.

**TODO: The following section needs to be written out!**


```
sku context sandstorm-rancher
sku mount-backup worker1
sku mount-backup worker2

cd ~/src/k8s/backup/worker1/*/codimd/
sku context [...]

sku restore clean-manifests -f config | kubectl apply -f - --dry-run=client
sku restore clean-manifests -f config | kubectl apply -f -
# wait for pods to be ready
sku ns [yourns]
kubectl get pods -w
sku restore mariadb  sql/.....mariadb.sql
sku restore persistentvolumes volumes
kubectl delete pods --all
# switch DNS
# switch deployment in .gitlab-ci.yml
# Check that everything works
```

