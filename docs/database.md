# Entering a Database via a Pod

We support entering a database by launching a `debug container`
next to a pod. So effectively, we create a **proxy inside the Kubernetes cluster** to access a database from the local machine.

**You need a recent Kubernetes >= 1.18, with [Debug Containers enabled](https://kubernetes.io/docs/tasks/debug-application-cluster/debug-running-pod/#ephemeral-container).**

This has the benefits that e.g. network policies do not interfere: If the Kubernetes pod is able to
access the database, so are we.

> **NOTE**: This section is **extremely opinionated** right now, fitting to the Sandstorm
> conventions. We'd love to refactor this to be more useful generally-purpose; please let us
> know what you need.

## Entering a MySQL database

First, ensure that you are in the correct namespace; and switch it if necessary
using `sku ns`.

By default, the database credentials are read from the Kubernetes cluster at the following locations:

- database host: read from a `db` ConfigMap, key `DB_HOST`
- database name: read from a `db` ConfigMap, key `DB_NAME`
- database user: read from a `db` ConfigMap, key `DB_USER`
- database password: read from a `db` Secret, key `DB_PASSWORD`

This fits well to how we at Sandstorm deploy applications.

Run the following command:

```bash
sku mysql cli
```

After a few seconds, you have an interactive MySQL CLI. You need
the `mysql` command line client installed.

### Support for mycli

[mycli](https://www.mycli.net/) is a more interactive command-line browser for MySQL. It can be installed on OSX by running `brew install mycli`.

After mycli is installed, you can run: 

```bash
sku mysql mycli
```

### Support for Sequel Ace

Only works for Mac OS X - you need [Sequel Ace](https://sequel-ace.com/) installed.

You can run:

```bash
sku mysql sequelace
```

### Support for Beekeeper Studio

You need [Beekeeper Studio](https://www.beekeeperstudio.io) installed.

You can run:

```bash
sku mysql beekeeper
```

Then, a connection string is printed. In Beekeeper Studio, press **Import from URL**,
and paste the connection string.




## Entering a Postgres database

First, ensure that you are in the correct namespace; and switch it if necessary
using `sku ns`.

By default, the database credentials are read from the Kubernetes cluster at the following locations:

- database host: read from a `db` ConfigMap, key `DB_HOST`
- database name: read from a `db` ConfigMap, key `DB_NAME`
- database user: read from a `db` ConfigMap, key `DB_USER`
- database password: read from a `db` Secret, key `DB_PASSWORD`

This fits well to how we at Sandstorm deploy applications.

Run the following command:

```bash
sku postgres cli
```

After a few seconds, you have an interactive Postgres CLI. You need
the `psql` command line client installed.

### Support for pgcli

[pgcli](https://www.pgcli.com/) is a more interactive command-line browser for Postgres. It can be installed on OSX by running `brew install dbcli/tap/pgcli`.

After pgcli is installed, you can run:

```bash
sku postgres pgcli
```

### Support for Beekeeper Studio

You need [Beekeeper Studio](https://www.beekeeperstudio.io) installed.

You can run:

```bash
sku postgres beekeeper
```

Then, a connection string is printed. In Beekeeper Studio, press **Import from URL**,
and paste the connection string.
