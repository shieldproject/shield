# Bug Fixes

* The Web UI, when sorting, is now case-insensitive.
* The data-system-specific storage footprint in the Web UI now no longer
  counts purged archives against the storage footprint.
* The core no longer leaks a SQL prepared statement when making requests
  to the SQLite3 backend, fixing an unbounded memory leak.
* The migration to database schema v12 now reports errors more granularly.
