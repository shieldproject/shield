This release chiefly introduces logic to existing and new database
schema migrations to fix fallout from the 8.6.0 release, in which
we mistakenly modified a historic migration to do something.

All database upgrade paths, including pre-8.6.0 → present,
(failing) 8.6.0 deployment → present, and brand new deployments,
should all work now.

# Improvements

- The `fs` plugin is now quieter by default, and will only turn on
  per-file debug logging if asked to do so via its own
  configuration.  This should greatly speed up backup operations
  on busy SHIELDs, since it reduces the database lock contention.

# Bug Fixes

- The front-end configuration wizard now properly looks up
  plugin metadata for an agent.  Previously, there was a
  Javascript variable shadowing bug that caused the front-end to
  return any arbitrary plugin metadata as the "correct" one.

- Task cancelation had a n inverted boolean assertion on global
  tenant-iness that has been fixed.
