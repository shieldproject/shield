# Improvements

- Check for the existence of the required top-level import
  manifest keys in `shield import`.

- The `s3` plugin can now be configured to use IAM instance
  metadata to assume roles inside of S3, instead of providing
  explicit access and secret key material.  Yay security!

- The `postgres` plugin now allows split read replica / write
  master backup and restore, for highly-available solutions.

# New Features

- Added `safe` target plugin for backing up and restoring Vault data.
