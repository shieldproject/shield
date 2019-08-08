# New Features

- Addition of new SHIELD plugin to backup/restore an `ETCD cluster`. We support different types of authentication for connection to the `ETCD cluster`. They include no authentication, only provide a trusted CA certificate, role-based authentication and certificate-based authentication. Our plugin can handle multiple client urls, so if an `ETCD node` goes offline it doen't effect the backup/restore functions. We can also backup/restore certain keys/values based on the key provided in the `prefix` field. The plugin can also perform a clean restore of the cluster using the `overwrite` field.

# Improvements

- The `cancel`, `task`, `restore-archive` and `purge-archive`
  commands in the SHIELD CLI now properly support short UUIDs,
  like all other commands.
