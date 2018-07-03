# Improvements

- Move the `vault.crypt` file out from under the `vault/` data
  directory sub-directory; that sub-directory is dedicated to the
  Vault instance, and we shouln't be putting other things in there.
