# Bug Fixes

- The `shieldd` binary now properly reports its release version in
  both CLI (`-v`) and web UI contexts.

- The archives list on the system page now no longer gives you the
  option of restoring invalid archives (i.e. purged stuff).
  Thanks @thomasmitchell for finding and reporting in #506.

# Improvements

- All `-v` handlers in CLI utilities now properly handle the 'dev'
  version as analogous to the empty ('') version, and revert to
  reporting the version of the binary as '(development)'.  This is
  mainly for packaing Docker images properly.

- All `shield*` CLI utilities, include the `shield` CLI itself,
  the `shieldd` daemon, and all helper binaries now sport options
  for getting their usage (`--help`) and versions (`--versions`).

- The `s3` plugin now accepts a URL as its `s3_host` endpoint
  parameter, affording operators more flexibility.
  The alternative was confusion!  See #509.

# Release Engineering

- Docker images can now be built with embedded release versions,
  for non-dev distribution as a container image.
