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
- [sku ns](#sku-ns)
- [sku enter](#sku-enter)
- [sku logs](#sku-logs)
- [sku encrypt (alpha)](#sku-encrypt)


### sku context

*This command allows to switch between different configured Kubernetes clusters*.

**List all configured contexts**: `sku context`

**Switch the active context**: `sku context [context-name]`


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

## Installation

The installation is done using homebrew:

```bash
brew install sandstorm/tap/sku
```

## Developing

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
