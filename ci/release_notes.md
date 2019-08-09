# New Features

- SHIELD now features a new `etcd` plugin for backing up and restoring your etcd key-value stores.  It supports single- and multi-node clusters and can authenticate via roles and X.509 certificates.  If you want, you can restrict the backup to a subset of the etcd tree (via a prefix setting).  It also supports _additive restore_ for situations that need it.

# Improvements

- The `cancel`, `task`, `restore-archive` and `purge-archive`
  commands in the SHIELD CLI now properly support short UUIDs,
  like all other commands.
