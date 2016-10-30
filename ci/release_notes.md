# Improvements

- The CLI now displays the store / target / schedule / retention
  policy names, alongside UUIDs in the Job creation TUI forms.
  Fixes #195.

- `shield show tasks` now honors the `--limit` parameter, so you
  don't have to look at all the tasks.  Fixes #164.

- `shield backend` now prints the currently selected backend.
  Fixes #194

- Both the Web UI and the CLI now lowercase plugin names before
  submitting them to the Rest API.  Fixes #156

- The `postgres` plugin once again allows users to customize the path
  to postgres binaries via the `pg_bindir` target configuration parameter.

  **NOTE** This re-introduces a feature that was previously removed, using
  the same configuration. Take care to insure that any targets you have
  configured are no longer specifying this key, as it was ignored until now.
  If it is specified, ensure that it is the correct value, or remove it, if
  it is unneeded. If it is specified and is an invalid value, your postgres
  backups will start to fail after upgrading.

# Bug Fixes

- Dropdowns on the Job Edit Form now remember the values the Job
  had before you opened the form.  Fixes #173

- The `mysql` plugin can now be used to backup _and_ restore all
  databases.  Fixes #211
