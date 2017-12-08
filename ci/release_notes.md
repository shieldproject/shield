# New Features

- The `azure` plugin now features a `path_prefix` setting to allow
  sharing of a single Azure Blobstore container amongst several
  jobs and/or SHIELDs.

# Improvements

- The `fs` plugin no longer relies on the `bsdtar` executable to
  function; instead, all tarball creation / extraction is handled
  directly by the plugin code, making it easier to deploy.

- The `test-store` and `purge` tasks that are scheduled in the
  slow loop are now skipped if the Vault is sealed.  This keeps
  the task list from growing with lots of tasks that will not be
  scheduled until later.  For `purge` tasks this wasn't a huge
  deal, but for `test-store` it meant that cloud storage would get
  slammed with test after test after test after test as soon as
  the SHIELD was unlocked.

# Breaking Changes

- The `fs` plugin no longer functions as a store plugin.  This
  configuration was deemed to dangerous in the wild, given the
  locality constraints.  If you need local-ish filesystem-backed
  storage, check out the `webdav` plugin.

# Bug Fixes

- WebSocket broadcast receivers are only registered _after_ a
  successful upgrade from plain HTTP to WebSockets, to avoid
  stalling out the core on badly-behaved clients.

- The CLI now honors `-k` everywhere it appears.

- It is now possible to update a target / store that was created
  without any configuration (no `--data` on create-*).

- CLI update-* commands now properly display the updated object
  attributes, instead of an empty report.

- The `create-auth-token` CLI command now honors `--json`.

- Fix javascript event handler stacking bugs in the web UI.  In
  short, form submissions would "remember" their previous onsubmit
  handlers, leading to some _very_ interesting errors on both
  client- and server-side.
