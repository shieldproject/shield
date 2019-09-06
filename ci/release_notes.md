# Improvements

- Cloud Storage detail pages in the web interface now show a
  timeline similar to the one shown for Data Systems, so that
  SHIELD operators have an easier time of troubleshooting failing
  storage configurations.

- The SHIELD CLI now displays task+log data for the last
  test-store task of a given store (for `shield store X` and
  `shield global-store Y`), to assist in troubleshooting failing
  storage configurations.

- Operators now have a web interface and some CLI commands for
  inspecting the state of SHIELD Data Fixups, and re-running them
  (if / when necessary).

- The `mongo` target plugin can now have options applied
  individually to just `mongodump` or `mongorestore`.

- Passwords and RSA private keys are now properly obscured in
  the web interface detail pages for both systems and cloud
  storage.  People without rights to see such credentials will
  still see the "REDACTED" string instead; but people with the
  required privilege will instead see the blurred-out obscured
  text that they can hover over to reveal.

- The `shield tasks` command can now filter down to only tasks
  that involve a particular tenant or global cloud storage system.

# Bug Fixes

- Data Fixups will now be properly skipped if they've already been
  applied.  Additionally, names / dates / summaries will be
  updated _every time_ the SHIELD Core boots up, to catch typos
  and mispellings there.

- The Data System detail page in the web interface no longer has a
  race condition between the start of an AJAX call for the plugin
  configuration details and a `shield:navigate` away from the
  page.  Other such race conditions involving AJAX should now also
  be fixed.

- Switching between tenants (with differing levels of access) now
  properly re-renders the sidebar to show your new privileges.
