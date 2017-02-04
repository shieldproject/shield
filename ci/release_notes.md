# Bug Fixes

- The `consul` plugin now lets you optional drop the username /
  password from your endpoint configuration, in-line with what the
  documentation said all along.

# Improvements

- The SHIELD CLI, `shield` no longer natters on incessently about
  which SHIELD backend it is going to use, which cleans up logs of
  our testing environment, and makes scripting easier.

- The `consul` plugin now defaults to `127.0.0.1:8500` for the
  `host` parameter, which is what most people were manually
  setting it to anyway.  The plugin is now truly 'zero-conf', Yay!
