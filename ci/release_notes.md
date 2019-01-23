# New Features

- The SHIELD Web UI now allows you to download the SHIELD CLI
  directly, for both MacOS (Darwin) and Linux.  From now on,
  SHIELD releases will include the paired version of the CLI.

- We now support minutely backups, but only from the CLI.

- New `shield op pry` for decrypting and inspecting the contents
  of a SHIELD Vault Crypt.

- New shield cli command 'delete-tenant' which will delete a tenant and clean up it's underlying configs with a -r 

# Improvements

- SHIELD now cleans up the Vault when archives are marked as
  expired (for purgation).

- Scheduled jobs no longer "stack" in the queue.  If SHIELD goes
  to schedule a backup and an existing task is in-flight for the
  same job, an already-cancelled task is stored in the database,
  as a placeholder to the task that should have run.

- Storage Health Check Tasks no longer stack.  SHIELD only allows
  one in-flight task for a given Cloud Storage System, at a time.

- The `shield` CLI now handles API endpoints with any number of
  trailing forward slash (`/`) characters.

- Update --help page on import to reflect correct roles

# Bug Fixes

- Web UI page dispatch logic now properly cancels all outstanding
  AJAX requests, to avoid a rather annoying lag/delay UX issue
  where pages would flip "back" to a previous node in the history,
  because a delayed AJAX request was still working away in the
  background.

- Updated go-s3 to help fix connection closing issue
