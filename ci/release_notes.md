# Improvements

- The `mongo` target plugin can now have options applied
  individually to just `mongodump` or `mongorestore`.

# Bug Fixes

- The Data System detail page in the web interface no longer has a
  race condition between the start of an AJAX call for the plugin
  configuration details and a `shield:navigate` away from the
  page.  Other such race conditions involving AJAX should now also
  be fixed.
