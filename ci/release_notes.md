# Bug Fixes

- The `consul` plugin now lets you optional drop the username /
  password from your endpoint configuration, in-line with what the
  documentation said all along.

- If an error is encountered trying to read the `pg_port` endpoint
  configuration directive in the `postgres` plugin, validation of
  the endpoint now fails properly.  Before, such cases (rare as they
  were) were being silently ignored.

- The `mysql` plugin now spells out what databases (even if that
  means "all") it is going to backup, when validation runs.  This
  parameter was missing from validation output, previously.

# Improvements

- The SHIELD CLI, `shield` no longer natters on incessently about
  which SHIELD backend it is going to use, which cleans up logs of
  our testing environment, and makes scripting easier.

- The `consul` plugin now defaults to `127.0.0.1:8500` for the
  `host` parameter, which is what most people were manually
  setting it to anyway.  The plugin is now truly 'zero-conf', Yay!

- The `mongo` plugin now defaults to `127.0.0.1` for the
  `mongo\_host` parameter, and `27017` for the `mongo\_port`
  property.  Similarly, authentication is optional, and will be
  skipped if you don't specify a username/password.  The plugin is
  now truly `zero-conf`, double-Yay!

- The `mysql` plugin now defaults to `127.0.0.1` for the
  `mysql\_host` parameter, leaving just the authentication
  parameters as required.

- The `s3` plugin now ignores all leading slashes on the `prefix`
  path, since they were interfering with properly retrieving
  archives from S3 for restore operations.
