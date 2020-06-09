# Improvements

- Docker images are now compiling via the go 1.13 toolchain.

- Agent Registration can now occur through chained load balancers,
  with standards-compliant comma-separated X-Forwarded-For
  headers.  Why you would want to do this is beyond me, but ¯\_(ツ)\_/¯

- The `metashield` plugin now trusts system X.509 Root CAs if no
  specific CA is supplied.

- Bootstrap restoration is simpler now, and the UI for init /
  restore is more streamlined.  See #680.
