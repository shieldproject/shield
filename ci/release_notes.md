# Improvements

- Move the `vault.crypt` file out from under the `vault/` data
  directory sub-directory; that sub-directory is dedicated to the
  Vault instance, and we shouldn't be putting other things in there.

- Threshold for storage now indicates the use of units in the form field, to
	prevent the accidental specification of 50 bytes when you meant 50 gigabytes.

- Improved results of the /v2/info and /v2/heath API endpoints to match its
  documented behaviour.

# Deprecations

- Removed FQDN from /v2/info as it was populated using DNS reverse lookups
	that were less than useful.

# Bug Fixes

- Storage health correctly stated during creation of ad-hoc runs.

- Scheduled jobs in timeline are not longer incorrectly as "Ad-hoc"

- Admin/Sessions page no longer shows all IP Addresses as `localhost` and
	shows the session creation time in human-readable format.

- Notes for targets are now displayed on the page for a given system.

- Errors encountered when unlocking the vault now notify the user.

- Release version correctly displayed on header instead of `(development)`
