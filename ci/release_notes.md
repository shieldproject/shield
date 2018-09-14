# New Features

- We now have a BackBlaze B2 storage plugin!

# Improvements

- Ordinals are now optional in monthly schedule specs (via the web
  UI), allowing front-end users to type '3' or '3rd', per their
  strongly-held personal preference.

- The `token` field of the `vault` plugins is now marked as a
  _password_, so that autocompletion in the browser gets turned off.
  Otherwise, Chrome/FF keeps wanting to leak your Vault tokens to
  people.

- The data directory and web UI root configurations are now
  properly validated by the SHIELD core.  If they do not exist,
  core startup is halted.  That way, you find out sooner if you've
  misconfigured something.  Wheee.

- `shield import` can now properly import fixed-key backup jobs.
  Just what the doctor ordered for BOSH and SHIELD backup and
  recovery.

# Bug Fixes

- Errors with hourly schedules are now properly handled and give a
  readable error message to the front-end.

- The `mysql` plugin can now properly restore a single database.

- Some silly typos (some copy-pasta, some bad whitespace, some
  we-don't-know-what-we-were-thinking) have been fixed in SHIELD
  CLI `--help` output.
