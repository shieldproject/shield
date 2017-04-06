# Improvements

- The `consul` plugin now supports SSL-based connections to Consul.
  Simply set your `host` to something beginning with `https://`.
  It also supports a new `skip_ssl_validation` plugin option, to ignore
  self-signed/invalid certs.

# Minor Cleanup

Removed unimplemented elasticsearch plugin
