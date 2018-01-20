# Bug Fixes

- Fix an egregious bug in the scheduling logic that was only
  considering jobs scheduled in the future to be "overdue".
  Since all jobs start out with a next_run of 0, this caused NO
  JOBS to ever be scheduled.  Thankfully, 8.x is still beta.

- Fix a segfault when dereferencing a nil Task during a broadcast.
  Now, we log that we got a nil task, to assist in tracking down
  why / where its occurring, rather than just crashing on panic.

- The `shield restore-archive` command now prints out the UUID of
  the task scheduled to run the restore, rather than the cryptic
  (and oh-so-unhelpful) string "%s!:bool=true"

- Neither `shield create-job`, nor `shield update-job` will allow
  you to create (or modify) jobs to have invalid, unparseable
  schedules.  This will keep the CLI from accidentally creating
  schedules that the web UI can't process.

# Improvements

- Global Storage Systems are available for selection during the
  backup configuration wizard in the web UI.

- Storage systems now properly report their health to all
  front-end views, fixing a few fixmes along the way.

# Developer Stuff

- `bin/testdev` now runs a WebDAV service on the nginx reverse
  proxy (on `$PORT+1`), since we can no longer use the `fs` plugin
  for storage operations.

  On MacOS, with homebrew, you'll want to reinstall nginx with
  WebDAV support: `brew reinstall --with-webdav nginx`
