# Improvements

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
