# Bug Fixes

- `shield-agent` will now propagate HTTP proxy environment
  variables: `http_proxy`, `https_proxy` and `no_proxy`, which
  some plugins (i.e. s3) can make use of.

- The `postgres` plugin no longer requires a host address.  If not
  specified, a local loopback (usually UNIX domain socket) will be
  attempted.

- The `postgres` plugin no longer requires a password.  If not
  specified, no authentication credentials will be sent.  This is
  usually paired with an empty (or missing) pg_host, to gain
  superuser access over loopback (given a 'trust' entry in HBA)
