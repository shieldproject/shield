# Improvements

- Submit buttons on forms now (a) disable themselves when clicked
  and (b) change their text to indicate an ongoing operation.
  This greatly increases the usability of the web UI.  See #505

- The default password for the failsafe account has been changed
  from `shield` to `password`, for more continuity across various
  packaging formats.  See #531

- The `shield tasks` command (and the backing API) can now filter
  tasks based on their task type (i.e. "backup", or "restore")
  See #523

- The `Encryption` column of the system detail page's backup jobs
  table now _always_ shows something.  For jobs that do not used
  the fixed key, the new tag is `randomized`.  See #536

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