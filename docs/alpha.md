# Alpha/Experimental Features


## sku encrypt

ALPHA: Encrypt Kubernetes credentials via Yubikey PIV module

You can encrypt the Client Credentials using your Yubikey's private key. This command
encrypts the keys and changes the Kubernetes config to decrypt the keys when you need them,
by setting up the "sku decryptCredentials" command as Exec Authentication Plugin for Kubectl.

As a user, this means that you always need to touch your Yubikey when issuing kubectl commands.

PREREQUISITES:
- install https://github.com/sandstorm/ykpiv-ssh-agent-helper in a recent version OR install
  OpenSC from https://github.com/OpenSC/OpenSC/releases
- ensure kubectl is installed in at least version 1.11.

*Examples:*

```
# set up encryption for the given context
# You need to tap your yubikey during setup; so that decryption can be properly tested.
	sku encrypt [context]
```

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

