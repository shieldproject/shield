# Improvements

- Shield will now only let a single task be run for a given
  target at a time. You can no longer run a backup and restore
  of the same data simultaneously.

# Bug Fixes

- Fixed an issue with the s3 plugin having issues when paths
  contained/started with double `/`s.
- Fixed an issue where purge tasks weren't properly created with the MySQL
  backend.
