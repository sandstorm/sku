# Connecting to NATS via a NatsUser

`sku nats` is a thin wrapper around the upstream [`nats` CLI](https://github.com/nats-io/natscli)
that authenticates as a `NatsUser` custom resource from the **current
Kubernetes namespace**.

> **Required to use `sku nats`:** the target cluster must run the
> [Sandstorm NatsAuthOperator](https://github.com/sandstorm/NatsAuthOperator).
> It owns the `NatsUser` CRD (`nats.k8s.sandstorm.de/v1alpha1`) and
> manages the per-user nkey secret (`<user>-nats-nkey`, key `nkey-seed`)
> and `status.connectionURLs` that `sku nats` reads from. Without the
> operator installed there are no `NatsUser` resources for `sku nats` to
> discover, so this command is a no-op.

You also need the upstream `nats` binary on your `PATH`
(`brew install nats-io/nats-tools/nats`).

## How it works

1. Lists `NatsUser` resources in the current namespace
   (`nats.k8s.sandstorm.de/v1alpha1`). If exactly one exists, it is used;
   otherwise you are prompted to pick one (or pass `--user NAME` to skip
   the prompt).
2. Reads the nkey seed from the operator-managed secret
   `<user>-nats-nkey`, key `nkey-seed`, into a `0600` tempfile that is
   shredded on exit (including on `Ctrl-C`).
3. Reads the server URL from `NatsUser.status.connectionURLs`, preferring
   `tls://` → `wss://` → `nats://` → `ws://`. Override with `--server URL`.
4. Execs `nats --nkey <tempfile> --server <url> <your args...>`.

`NATS_CONTEXT` is cleared in the child environment so a locally-configured
`nats context` cannot leak credentials.

## Usage

```bash
# Subscribe to everything as the only/selected NatsUser in this namespace
sku nats sub ">"

# Publish, skipping interactive user selection
sku nats --user sandstorm-admin pub foo bar

# Override the server URL (useful when status.connectionURLs is unset
# or you want to reach a different endpoint)
sku nats --server tls://your-server stream ls
```

Flags consumed by `sku` (everything else is forwarded to `nats`):

| Flag           | Meaning                                    |
|----------------|--------------------------------------------|
| `--server URL` | override server URL                        |
| `--user NAME`  | NatsUser name (skip interactive selection) |

## Autocompletion

`sku nats <TAB>` completes against the **real** `nats` CLI's own
completion script (rewired through a "Layer 2" dispatcher in the
generated `_sku`), so subjects, stream names, consumer names, etc. are
all completable just like running `nats` directly. Make sure you have
followed the [autocompletion setup](autocompletion.md) and that `nats`
is on `PATH` *at the time you (re)generated the completion script*.

If `nats` was not on `PATH` when the completion was generated, you fall
back to a static list of top-level subcommands (`pub`, `sub`,
`stream`, `consumer`, `kv`, `obj`, …). Regenerate the completion (see
[autocompletion](autocompletion.md)) after installing `nats` to get the
full passthrough behavior.
