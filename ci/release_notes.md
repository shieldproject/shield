# Improvements

- Submit buttons on forms now (a) disable themselves when clicked
  and (b) change their text to indicate an ongoing operation.
  This greatly increases the usability of the web UI.  See #505

- The web UI for rekeying SHIELD Core now correctly identifies
  when the operator would like to rotate the fixed key.  Also, the
  error messaging for an incorrect _current_ master password is
  better now, and by default, the "rotate fixed key" checkbox on
  the rekeying form is off.  See #546

- The default password for the failsafe account has been changed
  from `shield` to `password`, for more continuity across various
  packaging formats.  See #531

- The `shield tasks` command (and the backing API) can now filter
  tasks based on their task type (i.e. "backup", or "restore")
  See #523

- The `Encryption` column of the system detail page's backup jobs
  table now _always_ shows something.  For jobs that do not used
  the fixed key, the new tag is `randomized`.  See #536

- SHIELD now tracks when it last checked each agent separately
  from when it last "saw" the agent.  _Last Seen_ now means the
  point in time when the agent last connected to the SHIELD core,
  and _Last Checked_ is when the core last connected to the agent
  for metadata retrieval.

- SHIELD now allows agents to change their IP address; only the
  agent name is unchangeable.  Previously, attempts to change an
  agents registered IP address (without changing its name) would
  fail.

# Bug Fixes

- The MotD separator no longer displays if the MotD is empty
  or not specified.  See #530

- The Ad Hoc Backup and Restore wizards now handle the "empty"
  state more gracefully, and instead of showing an empty table
  when there are no data systems, they warn you that you have
  no systems to backup or restore.  See #532 and #533

- Stores (global and tenant-specific) can now be properly deleted
  via the web UI and CLI.

- When editing targets and stores on the webui changes are now
  persisted when editing again without a refresh.

- The "Agents of SHIELD" admin page no longer gets stuck in a
  loading loop whenever websocket events are seen.
