# Bug Fixes

- `shield-agent` will now propagate HTTP proxy environment
  variables: `http_proxy`, `https_proxy` and `no_proxy`, which
  some plugins (i.e. s3) can make use of.
