# Improvements

- The `postgres` plugin once again allows users to customize the path
  to postgres binaries via the `pg_bindir` target configuration parameter.

  **NOTE** This re-introduces a feature that was previously removed, using
  the same configuration. Take care to insure that any targets you have
  configured are no longer specifying this key, as it was ignored until now.
  If it is specified, ensure that it is the correct value, or remove it, if
  it is unneeded. If it is specified and is an invalid value, your postgres
  backups will start to fail after upgrading.
