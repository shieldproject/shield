# New Features

- The Web UI now provides feedback for failing jobs, on both the
  Dashboard (by way of a red banner), and on the Jobs page.
  Hopefully, this will make it easier to catch job failures before
  it's too late (i.e. when you need those backups that haven't been
  running properly...)

# Bug Fixes

- The scality plugin was incorrectly forcing http-based connections.
  It now correctly forces https-based connections.

# Improvements

- The SHIELD CLI's `--raw` mode now suports a new flag, `--fuzzy`,
  that enables fuzzy (inexact) name searching on the backend.

# Developer Stuff

- Moved the gspt stuff for process table credentials expunging
  into its own file that gets conditionally compiled only if CGO
  has been enabled.  This feature is not critical to the operation
  of plugins, and has caused some issues on OSX trying to
  cross-compile for Linux.  This fixes errors like the following:

  ```
  no buildable Go source files in $CODE/vendor/github.com/ErikDubbelboer/gspt
  ```

- SHIELD API tests are now excercised via the SHIELD CLI in
  `--raw` mode, which should help to both uncover bugs in the CLI,
  and also make it easier to augment and improve the SHIELD API.
