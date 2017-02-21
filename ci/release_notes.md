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
