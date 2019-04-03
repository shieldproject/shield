# Bug Fixes

- The `shieldd` binary now properly reports its release version in
  both CLI (`-v`) and web UI contexts.

# Improvements

- All `-v` handlers in CLI utilities now properly handle the 'dev'
  version as analogous to the empty ('') version, and revert to
  reporting the version of the binary as '(development)'.  This is
  mainly for packaing Docker images properly.

# Release Engineering

- Docker images can now be built with embedded release versions,
  for non-dev distribution as a container image.
