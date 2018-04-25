# Improvements

- Stop giving out bad advice in the help output of `create-job`
  and `update-job`, with respect to propery schedule syntax.

- Better error messaging in filepath walker when the `fs` plugin
  encounters an error or missing stat info.

- Credentials (as defined by plugins) are now redacted in task
  output, based on the rights of the viewer.

- The xtrabackup plugin now supports the `mysql_socket` option,
  for specifying the absolute path to the MySQL socket.


# Bug Fixes

- Properly set the job name and notes (summary) from the web ui
  wizard, instead of ignoring what the user provided.  Fixes #387.

- The `webdav` plugin no longer panics if you omit the `https://`
  or `http://` URL scheme from your DAV server URL.  Instead, it
  assumes HTTPS and keeps on truckin'.  Other URL parse errors are
  properly handled now as well.

- Handle people who don't set up their Github profiles properly,
  and are missing display names.  The Github API returns these as
  nil string pointers, instead of pointers to the empty string.
