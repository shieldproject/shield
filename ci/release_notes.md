# Improvements

- The `mongo` target plugin can now have options applied
  individually to just `mongodump` or `mongorestore`.

- Passwords and RSA private keys are now properly obscured in
  the web interface detail pages for both systems and cloud
  storage.  People without rights to see such credentials will
  still see the "REDACTED" string instead; but people with the
  required privilege will instead see the blurred-out obscured
  text that they can hover over to reveal.

# Bug Fixes

- The Data System detail page in the web interface no longer has a
  race condition between the start of an AJAX call for the plugin
  configuration details and a `shield:navigate` away from the
  page.  Other such race conditions involving AJAX should now also
  be fixed.
