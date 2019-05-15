# Bug Fixes

- The `shieldd` binary now properly reports its release version in
  both CLI (`-v`) and web UI contexts.

- The archives list on the system page now no longer gives you the
  option of restoring invalid archives (i.e. purged stuff).
  Thanks @thomasmitchell for finding and reporting in #506.

- System-initiated archive purges now properly set the store agent
  for purgation, so that the purge task has someone to talk to for
  removal of the archive from cloud storage.  See #514.

- The core scheduler now immediately fails any task for which the
  remote SHIELD agent does not signal a successful (rc=0) exit
  status.  This should clean up some task logs, and remove red
  herring issues like JSON unmarshal failures, while
  simultaneously ensuring that failed purge tasks are re-tried.
  See #518.

- Purge tasks are now being properly supplied with the restore key
  necessary for deleting the archive blob.  See #516.

- Agent Status tasks (op `agent-status`) were not previously being
  created with proper global tenant association.  This prohibited
  operators from viewing the details of those tasks.  We fixed
  this, and added a data fixup created to re-associate existing
  tasks.  See #522.

- The HUD now always registers the global cloud storage in its
  health data, so operators are aware of all issues with storaage
  systems that they might be using, global or tenant-private.
  See #504.

- Jobs created via the Web UI now properly set their "KeepN"
  attribute, which was missing from the ingestion / insertion.
  Accompanying this is a new data fixup that should re-calculate
  the `keep_n` database field wherever it is zero.  See #460.

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

- When purging archives manually, you can now supply
  human-friendly reasons for the purge.  For example, if the data
  is known to be bad in that particular vintage of the target
  system, you can purge the archives containing it, and explain
  that.  See #520

- Archives can now be annotated from the command-line, with the
  `annotate-archive` command.

- Manually purged archives now track their reason for purge as
  "manually purged", instead of "expired".  See #517.

- All system- and tenant-level objects can now be searched for,
  and referenced by short UUIDs.  This is huge (though short),
  going a long way to making the CLI easier to work with.

# Release Engineering

- Docker images can now be built with embedded release versions,
  for non-dev distribution as a container image.
