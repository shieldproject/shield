# New Features

- SHIELD now supports Fixed Key encryption for disaster recovery
  of backups for SHIELD itself.

# Improvements

- The `s3` plugin now uses pathd buckets, so it should work better
  with S3-workalikes that don't support DNS-style buckets.

- The `fs` plugin strips the base director from the files as they
  are archived, allowing archives to be portably replayed to
  different base directors on restore.

- The `mysql` and `xtrabackup` plugins are better now.

- `buckler import` works better now, no longer requiring a SHIELD
  core (via either `--core` or `$SHIELD_CORE`).  It also now
  supports skipping TLS verification of the SHIELD Core.

# Bug Fixes

- Plugins now accept boolish strings and numbers in place of
  actual booleans.

- Handle symlinks in the `fs` plugin

- The S3 plugin now properly sets a multipart upload chunk size
  of 5 MEGABYTES, not 5 GIGABYTES, so we don't OOM on VMs.  Oops.

- The WebUI can now display OAuth provider configuration (again).

- `buckler create-policy` now properly validates the expiry value
  as a number.

- SHIELD Core no longer leaks file descriptors when talking to the
  sealed Vaults.
