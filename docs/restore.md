# Restoring backups

> **NOTE**: This section is **extremely opinionated** right now, fitting to the Sandstorm
> conventions. We'd love to refactor this to be more useful generally-purpose; please let us
> know what you need.

## Cluster Setup
* Create Namespace `backup-job-downloads` in a rancher project where every user can read secrets
* In this namespace create a secret `backup-readonly-credentials` containing the following key/value pairs:
    * id_rsa -> the ssh key, registered on borgbase.com
    * for every node in the cluster add a key/value-pair for the borgbase repository url where the key format is repo_url_<node name>

## Local Setup (MacOS)
* Install osx fuse by
    * downloading from https://github.com/osxfuse/osxfuse/releases
    * execute installation
    * give allowance via Security -> Allow System Extensions -> Benjamin Fleischer
    * restart computer
* install borgbackup NOT via homebrew but manually:
    * download borg-macos64 from https://github.com/borgbackup/borg/releases
    * place it to `/usr/local/bin/borg`
    * make it executable via `chmod +x /usr/local/bin/borg`
    * test to execute `borg` in shell -> should give an error
    * Allow Borg via System Preferences -> Security
    * executing `borg` in shell should work now

## Restoring a backup

Done in 2 Steps:
1) Mount the backup from borgbase to your local system with sku mount-backup and
2) restore the desired data (config, volumes, databases) with the commands in sku restore

### Mount borgbase backup
* Prerequisites: see Local Setup
* Switch to the cluster you want the backups for with `sku context <clustername>`. You can check wich clusters are available with `sku context`.
* Get the name (! not the value of "node"-label like "worker1") of the node you want the backup for, e.g. `k3s2021-1` for our k3s2021 cluster.
* Execute `kubectl mount-backup <node name>` and enter the passphrase used to encrypt the backup. This passphrase should be found in Bitwarden (for our k3s2021 it is the one with many !!!)
* Now the finder should be opened, and you can browse the backup.
* The backups are mounted to ~/src/k8s/backup/...
* When finished, execute `sku umount-backup k3s2021-1` to unmount the backup!!

### Restore backups

There are several available commands to restore different data. Currently, our cluster node backups include: the kubernetes config (the yaml files for the resources), volumes and databases.

#### Restore Config
* In the mounted backup files, go to the directory for the namespace you want to restore the config for, e.g.: `cd ~/src/k8s/backup/worker1/*/codimd/`
* Change the cluster your sku points to (with sku context) to the desired cluster
* Since a) our clusters have operators and b) we want to test if the mechanisms to automatically create resources work, we don't want to apply all the resources in the backup as they are. 
  To only get the manifests we really need execute `sku restore clean-manifests -f config` and pipe it to kubectl apply like so: `sku restore clean-manifests -f config | kubectl apply -f - --dry-run=client` 
  or to actually execute`sku restore clean-manifests -f config | kubectl apply -f -`
* Wait for pods to be ready by checking with `sku ns <your namespace>` and `kubectl get pods -w`

#### Restore Databases
* Use `sku restore mariadb  sql/.....mariadb.sql`

#### Restore volumes
* Use `sku restore persistentvolumes volumes` where `volumes` is the mounted directory containing the volumes you want to restore
* This command starts a wizard helping you choose which directories to restore into which pods (the current files in the pod in the volume are removed, but backupped to your local machine)
* Actually, you could use this command to copy whatever files you like into the pods volumes.
