# Improvements

- Stop giving out bad advice in the help output of `create-job`
  and `update-job`, with respect to propery schedule syntax.

- Better error messaging in filepath walker when the `fs` plugin
  encounters an error or missing stat info.

# Bug Fixes

- Properly set the job name and notes (summary) from the web ui
  wizard, instead of ignoring what the user provided.  Fixes #387.
