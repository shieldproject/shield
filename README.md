[![Build Status](https://travis-ci.org/starkandwayne/shield.svg)](https://travis-ci.org/starkandwayne/shield)

S.H.I.E.L.D. Data Protection
============================

Questions? Join us in [Slack!](http://shieldproject.io/community/#slack)

![SHIELD Architectural Diagram](overview.gif)

What is SHIELD?
---------------

SHIELD is a data protection solution designed to make it easier for
operations to protect their critical infrastructural data.  It provides
primitives for scheduling automatic backups of key systems, including
PostgreSQL, MySQL, Consul, Redis and MongoDB, as well as a means for
restoring backups in the event of an outage.  Backups can be stored in a
variety of cloud providers, including S3, Scality, Microsoft Azure
Blobstore, and more.

Getting Started
---------------

The easiest way to get up and running with SHIELD is to deploy it via
[BOSH][bosh], using the [SHIELD Bosh Release][shield-bosh].

Backup (Target) Plugins
-----------------------

### `fs` - Local Filesystem Plugin

The `fs` plugin lets you back up arbitrary filesystem directories,
optionally filtering the set of protected files via an includes / excludes
system.

More information can be found
[here](https://godoc.org/github.com/starkandwayne/shield/plugin/fs).


### `postgres` - PostgreSQL Backup Plugin

Back up your PostgreSQL relational databases!  This plugin lets you back up
all databases (assuming you authenticate with an appropriately credentialed
pg account), or pick and choose what to backup.  Under the hood, this
leverages `pgdump`, a proven solution in the PostgreSQL world.

More information can be found
[here](https://godoc.org/github.com/starkandwayne/shield/plugin/postgres).

### `mysql` - MySQL Backup Plugin

Back up your MySQL relational databases!  This plugin lets you back up all
databases (assuming you authenticate with an appropriately credentialed
mysql account), or pick and choose what to backup.  This plugin leverages
`mysqldump`, which generates plain-text SQL backups, which can often be
replayed across MySQL versions.

More information can be found
[here](https://godoc.org/github.com/starkandwayne/shield/plugin/mysql).

### `xtrabackup` - MySQL XtraBackup Plugin

This plugin offers another way of protecting MySQL, using the `xtrabackup`
utility.

More information can be found
[here](https://godoc.org/github.com/starkandwayne/shield/plugin/xtrabackup).

### `cassandra` - Cassandra Backup Plugin

Back up Cassandra!

More information can be found
[here](https://godoc.org/github.com/starkandwayne/shield/plugin/cassandra).

### `consul` - Consul Backup Plugin

Back up the data stored in your Consul key-value store.

More information can be found
[here](https://godoc.org/github.com/starkandwayne/shield/plugin/consul).

### `mongo` - MongoDB Backup Plugin

Back up your MongoDB NoSQL database(s)!

More information can be found
[here](https://godoc.org/github.com/starkandwayne/shield/plugin/mongo).

Storage Plugins
---------------

### `s3` - Amazon S3 Storage Plugin

Store your encrypted backup archives in Amazon's Simple Scalable Storage
(S3) cloud.  All you need is a bucket, a key, and a secret, and you get
highly-available offsite archive storage.

More information can be found
[here](https://godoc.org/github.com/starkandwayne/shield/plugin/s3).

### `webdav` - WebDAV Plugin

If you can't make use of external, 3rd-party cloud storage for your backups,
but do have access to an HTTP/WebDAV server, you can use this storage plugin
to keep your archives there.

Note: often, use of the `webdav` plugin will compromise your disaster
survivability.  Make sure that your WebDAV store is sufficiently resilient
(HA, geographically dispersed, replicated, etc.), and that you aren't using
the same SHIELD core to back up your WebDAV store.

More information can be found
[here](https://godoc.org/github.com/starkandwayne/shield/plugin/webdav).

### `azure` - Microsoft Azure Storage Plugin

Store your encrypted backup archives in Microsoft's Azure Blobstore!

More information can be found
[here](https://godoc.org/github.com/starkandwayne/shield/plugin/azure).

### `google` - Google Cloud Storage Plugin

Store your encrypted backup archives in Google's Cloud!

More information can be found
[here](https://godoc.org/github.com/starkandwayne/shield/plugin/google).

### `swift` - OpenStack Swift Storage Plugin

Store your encrypted backup archives in your local OpenStack Swift blob
store!

More information can be found
[here](https://godoc.org/github.com/starkandwayne/shield/plugin/swift).
