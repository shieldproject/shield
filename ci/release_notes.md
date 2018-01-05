# Bug Fixes

- The v8 Web UI now properly renders target plugin forms, based
  on the metadata provided by the plugins themselves.  Previously,
  only the fs plugin was working, due to the next bug we fixed.

- The fs plugin was mistakenly reporting a store field, something
  that got missed when we removed its ability to act as a store
  plugin.

- The swift plugin now features field metadata.
