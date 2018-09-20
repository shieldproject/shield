# New Features

- The SHIELD Web UI now allows you to download the SHIELD CLI
  directly, for both MacOS (Darwin) and Linux.  From now on,
  SHIELD releases will include the paired version of the CLI.

- New `shield op pry` for decrypting and inspecting the contents
  of a SHIELD Vault Crypt.

# Bug Fixes

- Web UI page dispatch logic now properly cancels all outstanding
  AJAX requests, to avoid a rather annoying lag/delay UX issue
  where pages would flip "back" to a previous node in the history,
  because a delayed AJAX request was still working away in the
  background.
