# Improvements

- The Github OAuth provider now properly handles Github Enterprise
  for API work (user lookups, org lookups, etc.)

- The Github OAuth provider can now handle assignment across
  multiple tenants (including SYSTEM) from a single Github Org.

# Bug Fixes

- Fix a missing slash in the Github Authentication Provider
  display, in the administrative backend.
