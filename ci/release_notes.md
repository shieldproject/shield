# New Features

- Addition of new SHIELD plugin to backup/restore an `ETCD cluster`.

- Different authentication options include:
  - No Authentication
  - Provide a Trusted CA Certificate
  - Role-Based Authenticaion
  - Certificate-Based Authentication

- Can handle multiple client urls. If an `ETCD node` goes offline it doen't effect the backup/restore.

- Backup/restore certain keys/values based on the key provided in the `prefix` field.

- Can perform a clean restore of the cluster using the `overwrite` field.
