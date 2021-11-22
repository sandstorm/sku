# Alpha/Experimental Features

## sku rancher-backup

This command allows backing up a Rancher server, by fetching all API resources. The result is stored
in the current directory, and should be versioned in Git.

NOTE: it is *NOT* possible to directly import the Rancher config which has been exported this way,
but it can help to have a human-readable representation of the different resources; so that
it is traceable when/if some options have changed.

*Examples:*

```
sku rancher-backup --url https://your-rancher-server.de/v3 --token BEARER-TOKEN-HERE --output ./backup-directory
```

