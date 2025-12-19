# Entering a Kubernetes Pod as ROOT (via debug container)

Enter an interactive shell in a pod of **the current namespace**.
To select the pods you want to enter, you'll see a choice list if
there is more than one pod accessible.

```bash
sku debug
```

Optionally, you can restrict the pod list by specifying a label
selector:

```bash
sku debug app=foo
sku debug app=foo,component=app
```

We connect via `kubectl debug` and a sidecar `debug` container.
By default, we directly switch into the container's root file system.

## Debugging via extra Debug Containers

To end up in the debug container (e.g. if the target does not contain any kind of shell or tools),
you can run:

```bash
sku debug --no-chroot
```

Then, you will find the target container's data in `/proc/1/root`; and `ps` will list all processes.

By default, we use a `busybox` debug container; but you can also override this:

```bash

# our preferred debug container :-)
sku debug --no-chroot --image nicolaka/netshoot

# does NOT WORK right now, but would be nice to fix this container again :-)
sku debug --no-chroot --image docker-hub.sandstorm.de/public-containers/ebpf-tracer:latest
```

NOTE: This will take a bit to start, because the container image needs to be downloaded. Just be patient :-)