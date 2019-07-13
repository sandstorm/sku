# Sandstorm Kubernetes Tools

**Convenience tools to shorten long Kubectl calls**

This project provides a command line tool called `sku` to avoid long kubectl calls.

We provide convenience tooling to switch contexts and namespaces, enter containers, and many more.

## Installation / basic usage

To install the `sku` binary tool on OSX, we recommend to use homebrew:

```bash
brew install sandstorm/tap/sku
```

Then, use `sku help` to get inline help, or e.g. run `sku ns` to list all namespaces in your current Kubernetes cluster
(as configured in `~/.kube/config`).


## Features

The following commands are supported:

- [sku context](#sku-context)
- [sku add-config](#sku-add-config)
- [sku ns](#sku-ns)
- [sku enter](#sku-enter)
- [sku logs](#sku-logs)
- [sku encrypt (alpha)](#sku-encrypt-alpha)
- [sku rancher-backup (alpha)](#sku-rancher-backup-alpha)


### sku context

*This command allows to switch between different configured Kubernetes clusters*.

**List all configured contexts**: `sku context`

**Switch the active context**: `sku context [context-name]`


### sku add-config

This command allows adding an external kubeconfig file to the default ~/.kube/config;
so you can actually merge multiple Kubernetes configs together which you got from different sources.

*Example*:

```bash
sku add-config path-to-additional-kubeconfig-file
```

### sku ns

**List all namespaces**: `sku ns`

**Switch the active namespace**: `sku ns [namespace-name]`


### sku enter

Enter an interactive shell in a pod of the current namespace.
To select the pods you want to enter, you'll see a choice list.

Optionally, you can restrict the pod list by specifying a label
selector.

*Examples:*

```
# get presented a choice list which container to enter
	sku enter

# you can optionally specify a label selector to enter a specific pod.
# You cannot specify a pod name directly, as they change very often anyways.
	sku enter app=foo
	sku enter app=foo,component=app
```

### sku logs

Show the logs of a pod of the current namespace.
To select the pods you want to get the logs for, you'll see a choice list.

Optionally, you can restrict the pod list by specifying a label
selector.

*Examples:*

```
# get presented a choice list which logs to show
	sku logs

# you can optionally specify a label selector to show only the specific logs
# You cannot specify a pod name directly, as they change very often anyways.
	sku logs app=foo
	sku logs app=foo,component=app
```


### sku encrypt (alpha)

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

### sku rancher-backup (alpha)

This command allows backing up a Rancher server, by fetching all API resources. The result is stored
in the current directory, and should be versioned in Git.

NOTE: it is *NOT* possible to directly import the Rancher config which has been exported this way,
      but it can help to have a human-readable representation of the different resources; so that
      it is traceable when/if some options have changed.

*Examples:*

```
sku rancher-backup --url https://your-rancher-server.de/v3 --token BEARER-TOKEN-HERE --output ./backup-directory
```


## Developing

Simply have a modern Go version installed; check out the project somewhere (NOT in $GOPATH, as we use Go Modules),
and then run `./build.sh`.

## Releasing new versions


### Prerequisites for releasing

1. ensure you have [goreleaser](https://goreleaser.com/) installed:

  ```bash
  brew install goreleaser/tap/goreleaser
  ```

2. Create a new token for goreleaser [in your GitHub settings](https://github.com/settings/tokens); select the `repo` scope.

3. put the just-created token into the file `~/.config/goreleaser/github_token`



### Doing the release

Testing a release:

```
goreleaser --snapshot --skip-publish --rm-dist --debug
```

Executing a release:

1. Commit all changes, create a new tag and push it.

```
TAG=v0.9.0 git tag $TAG; git push origin $TAG
```

2. run goreleaser:

```
goreleaser --rm-dist
```


## Similar Projects

- [kubectx and kubens](https://github.com/ahmetb/kubectx/) to switch the Kubernetes context and namespace
