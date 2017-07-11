## New Features

- You can now pass arbitrary mongo options to the `mongodump`
  command-line utility in `mongo` plugin (Thanks @skburgart!)

- New `consul-snapshot` plugin should allow you to backup a modern
  Consul via the snapshotting system, rather than a walk of the
  key-value tree.

- New `swift` storage plugin for backing up to OpenStack Swift /
  Rackspace Cloud Files endpoints.

## Bug Fixes

- The SHIELD CLI got a new `--update-if-exists` flag to update
  matching _things_ instead of creating new, identical ones.
