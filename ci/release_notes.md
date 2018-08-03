# New Features

- Compression of archives is now optional, on a per-target basis.
  If you have really big databases and don't want to bother compressing
  them, you can now turn that off and get done with your data protection
  tasks sooner!

- The Tasks API now has new time boundary range parameters, for retrieving
  tasks based on when they started and/or stopped.

# Improvements

- The Systems and Storage views now have the ability to toggle between a
  card-based layout (the default), and a table layout.

- Tags in the Systems View Timeline are now only shown for non-backup tasks,
  and only for successfully completed tasks.  In practice, this means that
  restore operations get tags and no one else does.

- The Retention Policy API / UI / CLI is better.  Namely, the API matches
  the documation (it's a PATCH not a PUT), and we have proper bounds
  checking on expiry days and policy name lengths.

# Bug Fixes

- When restoring archives with the CLI, and targeting a different data
  system than the archive originally came from, everything works as
  expected.
