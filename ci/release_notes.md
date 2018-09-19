# New Features

- New `shield op pry` for decrypting and inspecting the contents
  of a SHIELD Vault Crypt.

# Bug Fixes

- Web UI page dispatch logic now properly cancels all outstanding
  AJAX requests, to avoid a rather annoying lag/delay UX issue
  where pages would flip "back" to a previous node in the history,
  because a delayed AJAX request was still working away in the
  background.
