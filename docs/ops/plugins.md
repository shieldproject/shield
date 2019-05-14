Plugins Reference
=================

Target Plugins
--------------

Target plugins are used to perform backup and restore operations against
actual data systems like PostgreSQL, MySQL, and Consul.

###  Local Filesystem

The `fs` plugin lets you back up and restore files and directories on
locally attached (or locally-accessible) disks.  It performs backups by
creating a POSIX _tar_ archive of the selected file set, which it unpacks on
restore.  No compression is used directly by the plugin.

**Note:** In previous versions of SHIELD (notably 7.x and below), the `fs`
plugin could also be used as a storage plugin.  This caused large amounts of
confusion, and often lead to data loss.  As of SHIELD 8.x, this behavior has
been removed outright.

#### Configuration Options

- `base_dir` **(required)** - The base directory to back up.  All files and
  directories underneath this directory will be included in the backup
  archive.  During restore, the files in the backup archive will be unpacked
  to this directory (which might be different from where they were
  originally  backed up).

  This option must be present, and must be an _absolute_ path.

- `include` - A pattern for determining what files to include in the backup
  archive.  Anything not matching this pattern will be left out.
  This has no effect during restore; all files in the backup archive will be
  extracted.

  By default, all files are included (pursuant to `exclude`).

- `exclude` - A pattern for determining what files _not_ to include in the
  backup archive.  Anything matching this pattern will be left out.
  This has no effect during restore; all files in the backup archive will be
  extracted.

#### Matching Files for `include` and `exclude`

The `include` and `exclude` options let you selectively include and exclude
files from the backup archive.  These options each take a single _shell
glob_ which will be matched against each file name.

Shell globs have the following syntax:

    *      Match zero or more characters.
    ?      Match zero or one character.
    [a-z]  Match a range of characters, in this example,
           lowercase alphabetic characters.
    [^a-z] Match anything that doesn't match [a-z]
    [xaby] Match either 'x', 'a', 'b', or 'y'
    \*     Match a literal asterisk.
           Also works for '?', '[', and ']'

For example, the file name `/var/data/index/foo.db` would match all of the
following globs:

  1. `foo.*`
  2. `*` (but everything matches this so...)
  3. `*.db`
  4. `foo?.[a-f]*`

It does not match the following globs:

  1. `FOO.*` - globs are case sensitive.
  2. `*/index/*` - matching is not applied to the path.
  3. `(foo|bar).*` - globs are not regular expressions.

The `fs` plugin evaluates exclusions and then inclusions.  This makes it
easy to handle the 80% case of _back up everything except the *.tmp files_.
By default, nothing is excluded and everything is included, so if you know
specifically what you want to back up, you can specify just the `include`
pattern.

Here's a few illustrative examples (`base_dir` assumed present, but not
shown):

  - Only back up the PDF documents

        { "include": "*.pdf" }

  - Don't bother backing up the binary files

        { "exclude": "*.bin" }

  - Back up all the text files, unless they start with underscores
    (a primitive form of file hiding)

        { "include": "*.txt",
          "exclude": "_*" }


### PostgreSQL Database

The `postgres` plugin lets you back up and restore [PostgreSQL][postgres]
databases.  It uses standard PostgreSQL tooling for backups: `pg_dump` and
`pg_dumpall` for backups, and `psql` to restore.

[postgres]: https://www.postgresql.org/

During restore, connected clients will be forcibly disconnected so that the
databases they were using can be dropped and recreated.  This happens
transparently, but does require that client software be able to gracefully
reconnect as needed.

#### Configuration Options

- `pg_host` - The IP address or host name of the PostgreSQL database server
  to target.  If this is not set, the plugin will connect _locally_, using
  the UNIX domain socket (usually somewhere in `/var`).

  **Note:** If you explicitly want to connect over loopback via TCP, you
  have to explicitly set `pg_host` to 127.0.0.1.

- `pg_port` - The port that the PostgreSQL database software is listening to
  for client connections.  Defaults to `5432`, the standard PostgreSQL port.

- `pg_user` - The username to authenticate as.  If you leave this blank, the
  UID of the SHIELD agent process will be used.  For BOSH deployments, this
  is usually `vcap`, which is generally what you want for local agents.

- `pg_password` - The password to authenticate as.  For local agents
  (connecting over loopback TCP, or via the UNIX domain socket) you can
  often leave this blank, especially in most BOSH deployments.

- `pg_database` - The name of a single database to back up.
  By default, all databases will be included in the backup archive.

- `pg_read_replica_host` - The IP address or host name of a read replica
  database server.  If specified, back up operations will be carried out
  against this host, instead of `pg_host`.

- `pg_read_replica_port` - The TCP port that the read replica is listening
  on.  This has no effect if `pg_read_replica_host` is not set.  It defaults
  to the value of `pg_port`.

- `pg_bindir` - The path to the directory which contains the PostgreSQL dump
  utilities, and psql.  Defaults to `/var/vcap/packages/postgres-9.4/bin`.
//If not specified, the plugin will attempt to find `pgdump`, `pg_dumpall`, and `psql` via the agent's plugin paths setting.

#### Co-locating the SHIELD PostgreSQL Addon

If you are running the SHIELD agent on a different machine than your PostgreSQL
database server, you will need to install the [SHIELD PostgreSQL Addon][shield-postgres]
to get the pg\_dump, pg\_dumpall and psql tools.

Add the release to the top-level `releases:` section in your agent
deployment manifest:

    releases:
      - name:    shield-addon-postgres
        version: latest   # or a specific version

Then add the job that corresponds to the version of PostgreSQL you want to
backup:

    instance_groups:
      - name: your-shield-agent
        jobs:
          - release: shield-addon-postgres
            name:    shield-addon-postgres-10.1

Check the [add-on's README][shield-postgres] for full details of available
versions.


### MySQL / MariaDB

#### Backing Up Via `mysqldump`

The `mysql` plugin lets you back up and restore [MySQL][mysql] /
[MariaDB][mariadb] databases, using the pure-SQL `mysqldump` utility.  For
very large database, the time spent serializing database records into SQL
commands may be prohibitive, and you may want to investigate use of the
`xtrabackup` plugin instead.

[mysql]:   https://www.mysql.com/
[mariadb]: https://mariadb.org/

Restores are handled by feeding the SQL backup through the standard `mysql`
utility.  Connected clients will remain connected during restore.

##### Configuration Options

- `mysql_host` - The IP address or host name of the MySQL database server to
  target.  This defaults to `127.0.0.1`, but can be changed to allow for
  remote backup/restore, from a network-attached SHIELD agent.

- `mysql_port` - The TCP port to connect to, which the database process
  should be listening to.  Defaults to `3306`, the standard MySQL port.

- `mysql_user` **(required)** - The username to authenticate with.

- `mysql_password` **(required)** - The password for the `mysql_user`.

- `mysql_read_replica` - The IP address or host name of a read replica
  database server.  If specified, back up operations will be carried out
  against this host, instead of `mysql_host`.

- `mysql_database` - The name of a single database to back up.
  By default, all databases will be included in the backup archive.

  **Note:** this setting has _no effect_ on restore -- whatever is included
  in the backup archive will be restored.

- `mysql_bindir` - The path to the directory which contains the `mysql` and
  `mysqldump` utilities.
  Defaults to `/var/vcap/packages/shield-mysql/bin`, which works well with
  the [SHIELD MariaDB Addon][shield-mysql].
//If not specified, the plugin will attempt to find both `mysqldump` and `mysql` via the agent's plugin paths setting.

- `mysql_options` - A set of `mysqldump` command-line flags.
  Refer to the MySQL documentation to see what you can set.

#### Backing Up Via `xtrabackup`

The `xtrabackup` plugin lets you back up and restore your [MySQL][mysql] /
[MariaDB][mariadb] databases using a filesystem-based approach leveraging
[Percona's XtraBackup][xtrabackup] utility.  This method is often faster, if
less portable, than the pure SQL approach taken by the `mysql` plugin.

[xtrabackup]: https://www.percona.com/software/mysql-database/percona-xtrabackup

**Note:** Since xtrabackup requires access to the MySQL / MariaDB data
directory, it can only be run from the database host itself.

##### Configuration Options

- `mysql_user` **(required)** - The username to authenticate with.

- `mysql_password` **(required)** - The password for the `mysql_user`.

- `mysql_socket` - The path to the MySQL UNIX domain socket.
  Defaults to `/var/vcap/sys/run/mysql/mysqld.sock`.

- `mysql_databases` - A comma-separated list of databases to back up.
  By default, all databases are included in the backup archive.

  **Note:** this setting has _no effect_ on restore -- whatever is included
  in the backup archive will be restored.

- `mysql_datadir` - The path to the directory which contains the MySQL data
  files.  The SHIELD agent (effective) user needs read access to this
  directory for backups, and write access for restores.

  Defaults to `/var/lib/mysql`

- `mysql_temp_targetdir` - The path to a temporary filesystem work space for
  the xtrabackup tool to work.  This directory must be empty, and the
  underlying filesystem must be at least as large as the complete MySQL
  database(s) being backed up.

  Defaults to `/tmp/backups`.

- `mysql_xtrabackup` - The path to the `xtrabackup` binary.
  Defaults to `/var/vcap/packages/shield-mysql/bin/xtrabackup`, which works
  well with the [SHIELD MariaDB Addon][shield-mysql].
//If not specified, the plugin will attempt to find `xtrabackup` via the agent's plugin paths setting.

- `mysql_tar` - The path to the `tar` binary.
  If not specified, the plugin will attempt to find `tar` via the agent's
  plugin paths setting.


#### Co-locating the SHIELD MariaDB Addon

If your MySQL installation does not include xtrabackup, you will need to
install the [SHIELD MariaDB Addon][shield-mysql].

Add the release to the top-level `releases:` section in your agent
deployment manifest:

    releases:
      - name:    shield-addon-mysql
        version: latest   # or a specific version

Then add the job that corresponds to the version of MySQL or MariaDB you
want to backup:

    instance_groups:
      - name: your-shield-agent
        jobs:
          - release: shield-addon-mysql
            #name:   shield-addon-xtrabackup-2.4

Check the [add-on's README][shield-mysql] for full details of available
versions.


### MongoDB NoSQL

The `mongo` plugin lets you back up and restore your [MongoDB][mongodb]
databases.  It relies on the `mongodump` and `mongorestore` utilities,
installed on the agent host.

[mongodb]: https://www.mongodb.com/

#### Configuration Options

- `mongo_host` - The IP address or host name of the MongoDB installation to
  target.  This defaults to `127.0.0.1`, but can be changed to allow for
  remote backup/restore, from a network-attached SHIELD agent.

- `mongo_port` - The TCP port to connect to, which the `mongod` process
  should be listening to.  Defaults to `27017`, the standard MongoDB port.

- `mongo_user` - The name of the user account to authenticate to MongoDB as,
  for performing both backup and restore tasks.  If not specified, no
  authentication is performed.

- `mongo_password` - The password to use when authenticating with
  the `mongo_user` account.

- `mongo_database` - The name of a specific database to back up.
  By default, all databases will be included in the backup archive.

  **Note:** This setting has _no effect_ on restore -- whatever is included
  in the backup archive will be restored.

- `mongo_bindir` - The path to the directory which contains the dump and
  restore binaries.
  Defaults to `/var/vcap/packages/shield-mongo/bin`, which works well with
  the [SHIELD MongoDB Add-on][shield-mongo].
//If not specified, the plugin will attempt to find both `mongodump` and `mongorestore` via the agent's plugin paths setting.

- `mongo_options` - A set of arbitrary command-line flags to pass to the
  dump and restore tools.  For example, `--ssl` will enable SSL/TLS when
  communicating with MongoDB.  Refer to the MongoDB documentation for more
  details.


#### Co-locating the SHIELD MongoDB Add-on

If you are running the SHIELD agent on a different machine than your MongoDB
database server, you will need to install the [SHIELD MongoDB Add-on][shield-mongo]
to get the mongodump and monogrestore tools.

Add the release to the top-level `releases:` section in your agent
deployment manifest:

    releases:
      - name:    shield-addon-mongodb
        version: latest   # or a specific version

Then add the job that corresponds to the version of MongoDB you want to
backup:

    instance_groups:
      - name: your-shield-agent
        jobs:
          - release: shield-addon-mongodb
            name:    shield-addon-mongo-tools-3.4

Check the [add-on's README][shield-mongo] for full details of available
versions.


### Consul Key-Value Store

The `consul` plugin lets you back up a [Consul][consul] key-value store.
It works by walking the entire store and capturing the value of every key
in the hierarchy.

[consul]: https://www.consul.io/

On restore, keys from the backup archive will be put back into the running
key-value store, but existing keys will _not_ be removed.


#### Configuration Options

- `host` - The full HTTP(S) URL of the Consul to back up.
  Defaults to `http://127.0.0.1:8500`.

- `username` - The username to authenticate to Consul (via HTTP Basic
  Authentication).  By default, no authentication is performed.

- `password` - The password for the given `username`.

- `skip_ssl_validation` - Whether to verify the X.509 certificate of the
  Consul API, or not.  If your Consul host is using an untrusted, expired,
  or self-signed certificate, you can set this option to true to bypass
  verification failure.  **_This is not recommended_** for production use.


### CF Service Brokers

#### Dockerized PostgreSQL CF Service Broker

The `docker-postgres` plugin lets you back up data in a Dockerized
PostgreSQL service broker that uses the cf-containers-broker provided by the
docker-boshrelease
(<https://github.com/cloudfoundry-incubator/docker-boshrelease>).

This is a highly specific configuration, and this plugin does not work with
other PostgreSQL + Docker combinations.

**Note**: This plugin must be executed on the system that runs the container
broker, which requires a co-located SHIELD agent in almost all cases.

##### Configuration Options

_This plugin has no configuration options._

##### How It Works (Implementation Details)

Backups are performed by connecting to the local Docker daemon, and finding
all the running containers.  It then iterates over that list, compiles some
metadata about port mappings (from Docker), and executes a `pg_dump` backup
on the running PostgreSQL instance.  All of this information is then stored
in a POSIX _tar_ archive.

Restore operations work through the tar archive, bringing up a new Docker
container for each entry, remapping ports, and restoring the database.
This is a service-affecting operation, since any existing container images
will be destroyed, and any connected clients will be disconnected.


#### Cloud Foundry RabbitMQ Service Broker

The `rabbitmq-broker` plugin lets you back up the configuration of the Cloud
Foundry RabbitMQ Service Broker
(<https://github.com/pivotal-cf/cf-rabbitmq-release>).  However, given that
it only uses stock RabbitMQ API calls, it may work with other
configurations.

**Note**: RabbitMQ is usually used as a non-durable message queue and
dispatch / routing system.  As such, this plugin only backs up the
_metadata_ of a RabbitMQ brokered installation, not the actual messages.

This plugin uses the RabbitMQ Management API, which requires that the
`rabbitmq_management` plugin be loaded into your RabbitMQ installation.

##### Configuration Options

- `rmq_url` **(required)** - The HTTP(S) URL of the RabbitMQ Management API,
  which usually runs on port 15672.

- `rmq_username` **(required)** - The username to authenticate to the
  management API as.

- `rmq_password` **(required)** - The password for the `rmq_username`
  account.

- `skip_ssl_validation` - Whether or not to bypass the validation of the
  X.509 certificate presented by the management API.  This is **not
  recommended for production use!**


#### Cloud Foundry Redis Service Broker

The `redis-broker` plugin lets you back up the configuration of the Cloud
Foundry Redis Service Broker
(<https://github.com/pivotal-cf/cf-redis-release>).  It is _not_ a
general-purpose Redis backup plugin, and will not work with stock Redis
installations.

**Note**: Since Redis does not allow backups across the network, any target
using this plugin must execute on a co-located SHIELD agent, on the Redis VM.

##### Configuration Options

- `redis_type` **(required)** - The type of Redis VM being targeted.  Must
  be one of either `dedicated` or `shared`.


##### How It Works (Implementation Details)

A Redis Service Broker deployment features two types of VMs: shared and
dedicated.  A shared VM runs multiple Redis processes, each bound on their
own port, with their own append-only (AOF) file.  A dedicated VM runs a
single Redis process (usually on the standard port).

The plugin backs up all data in `/var/vcap/store`, which works regardless of
the type of Redis VM being targeted.

How the plugin restores data depends on the type of Redis VM.

For shared VMs, a restore will stop the service broker process, terminate
all Redis instances, and extract the backup archive back into
`/var/vcap/store`.  Then, it validates the AOF and resolves any corruption
issues that may have occurred during backup (like a mid-write snapshot).
Finally, it restarts the service broker process, which will then re-launch
all of the configured Redis instances.

For dedicated VMs, a restore will stop the service broker's agent process,
and the Redis instance itself, and extract the backup archive back into
`/var/vcap/store`.  Then it validates the AOF and resolves any corruption
issues that may have occurred during backup (like a mid-write snapshot).
Finally, it restarts the agent and Redis process.

Restore operations are service-impacting, as the Redis instances are
shut down for the duration of the restore. Additionally, the service broker
apparatus is disabled, to prevent creation of new service instances during
the restoration.


### BOSH Backup / Restore

#### BOSH Backup / Restore (for deployments)

The `bbr-deployment` plugin lets you back up a BOSH deployment using
[BBR][bbr].

[bbr]: https://github.com/cloudfoundry-incubator/bosh-backup-and-restore

##### Configuration Options

- `bbr_target` **(required)** - The IP address or host name of your BOSH
  director.

- `bbr_deployment` **(required)** - The name of the deployment to protect.

- `bbr_username` **(required)** - The username to use when authenticating to
  the BOSH director.

- `bbr_password` **(required)** - The password for `bbr_username`.

- `bbr_cacert` **(required)** - The X.509 certificate of the Certificate
  Authority that signed the BOSH director's certificate.

- `bbr_bindir` - The path to the directory which contains the `bbr`
  executable.
  Defaults to `/var/vcap/packages/bbr/bin`, which works well with the
  [SHIELD BBR Addon][shield-bbr].
//If not specified, the plugin will attempt to find `bbr` via the agent's plugin paths setting.

##### Co-locating the SHIELD BBR Addon

If you are running the SHIELD agent on a machine that does not provide the
`bbr` executable, you will need to install the
[SHIELD BBR Addon][shield-mongo].

Add the release to the top-level `releases:` section in your agent
deployment manifest:

    releases:
      - name:    shield-addon-bbr
        version: latest   # or a specific version

Then add the job to install `bbr`:

    instance_groups:
      - name: your-shield-agent
        jobs:
          - release: shield-addon-bbr
            name:    bbr

Check the [add-on's README][shield-bbr] for more information.


#### BOSH Backup / Restore (for directors)

The `bbr-director` plugin lets you back up a BOSH director using [BBR][bbr].

##### Configuration Options

- `bbr_host` **(required)** - The IP address or host name of your BOSH
  director.

- `bbr_sshusername` **(required)** - The username to use when SSHing into
  the BOSH director VM to execute the BBR backup / restore operation.

- `bbr_privatekey` **(required)** - The SSH private key that has been
  authorized for use by `bbr_sshusername`.

- `bbr_bindir` - The path to the directory which contains the `bbr`
  executable.
  Defaults to `/var/vcap/packages/bbr/bin`, which works well with the
  [SHIELD BBR Addon][shield-bbr].
//If not specified, the plugin will attempt to find `bbr` via the agent's plugin paths setting.


##### Co-locating the SHIELD BBR Addon

If you are running the SHIELD agent on a machine that does not provide the
`bbr` executable, you will need to install the
[SHIELD BBR Addon][shield-mongo].

Add the release to the top-level `releases:` section in your agent
deployment manifest:

    releases:
      - name:    shield-addon-bbr
        version: latest   # or a specific version

Then add the job to install `bbr`:

    instance_groups:
      - name: your-shield-agent
        jobs:
          - release: shield-addon-bbr
            name:    bbr

Check the [add-on's README][shield-bbr] for more information.



Storage Plugins
---------------

### Amazon S3

The `s3` plugin lets you store backup archives in an [Amazon AWS Simple
Storage Service][s3] bucket.  In theory, this plugin should also work with
other implementations, not from Amazon, which we term _S3 work-alikes_.

[s3]: https://aws.amazon.com/s3/

Backup archives will be stored in a file name / path that encodes the date
and time of the backup operation, to make it easier to track down a specific
archive later:

    $prefix/YYYY/MM/DD/YYYY-MM-DD-HHmmSS-$uuid

for example, given a `prefix` of "prod/backups", a backup might be stored
at:

    prod/backups/2018/07/12/2018-07-12-134255-f3b564f2-ef62-4e38-9d94-ba17c37abf09


#### Configuration Options

- `access_key_id` **(required)** - The AWS Access Key ID to use for
  authenticating to S3.  For Amazon, this usually starts with "AKI".

- `secret_access_key` **(required)** - The secret key that corresponds to
  the access key ID.

- `bucket` **(required)** - The name of the S3 bucket to store backup
  archives in.

- `prefix` - An optional prefix for backup archive paths.  This can be
  useful if you are sharing a bucket between multiple teams, or across
  two or more different environments, and want to be able to keep them
  separate for out-of-band retrieval.

  By default, no prefix will be used.

- `s3_host` - Override the Amazon S3 backend endpoint.  This is _required_
  if you wish to use an S3 work-alike.
  Defaults to `s3.amazonaws.com`.

- `s3_port` - Override the TCP port of the S3 work-alike backend.
  Defaults to `443`.

- `skip_ssl_validation` - Whether to verify the X.509 certificate of the S3
  backend endpoint, or not.  If your local S3 work-alike is using an
  untrusted, expired, or self-signed certificate, you can set this option to
  true to bypass verification failure.  **_This is not recommended_** for
  production use.

- `part_size` - The multipart upload size.  Amazon S3 proper uses variable
  multipart sizes, but some work-alikes require this to be set to specific
  values.

- `signature_version` - All S3 protocol requests include a header signature
  to validate and verify each request.  The protocol supports two different
  methods of signature generation, version 2 and version 4.

  For Amazon S3 proper, version 4 should be used.  Some work-alikes,
  however, only support version 2.

  Defaults to version 4.

- `socks5_proxy` - A SOCKS5 proxy endpoint URL to use for tunneling all
  traffic to and from the S3 backend.  By default, no proxy is used.

### Google Cloud Storage

The `google` plugin lets you store backup archives in [Google Cloud's
blobstore storage system][gcs], which conceptually behaves a lot like
Amazon's S3.

[gcs]: https://cloud.google.com/storage/

Backup archives will be stored in a file name / path that encodes the date
and time of the backup operation, to make it easier to track down a specific
archive later:

    $prefix/YYYY/MM/DD/YYYY-MM-DD-HHmmSS-$uuid

for example, given a `prefix` of "prod/backups", a backup might be stored
at:

    prod/backups/2018/07/12/2018-07-12-134255-f3b564f2-ef62-4e38-9d94-ba17c37abf09


#### Configuration Options

- `bucket` **(required)** - The name of the GCS bucket to store backup
  archives in.

- `json_key` - The full GCE service account key (a JSON string form, for
  authenticating to Google Cloud.  This is _required_ if the SHIELD agent
  is not running from a Google Compute Engine VM, or if you want to use
  different GCE IAM credentials for storage than you do for VM deployment.

- `prefix` - An optional prefix for backup archive paths.  This can be
  useful if you are sharing a bucket between multiple teams, or across
  two or more different environments, and want to be able to keep them
  separate for out-of-band retrieval.

  By default, no prefix will be used.


### Microsoft Azure

The `azure` plugin lets you store backup archives in [Microsoft Azure's
Blobstore][azure-bs], in a storage container.

[azure-bs]: https://azure.microsoft.com/en-us/services/storage/blobs/

Backup archives will be stored in a file name / path that encodes the date
and time of the backup operation, to make it easier to track down a specific
archive later:

    $prefix/YYYY-MM-DD-HHmmSS-$uuid

for example, given a `prefix` of "prod/backups", a backup might be stored
at:

    prod/backups/2018-07-12-134255-f3b564f2-ef62-4e38-9d94-ba17c37abf09


#### Configuration Options

- `storage_account` **(required)** - The name of the Azure Storage Account
  to use when accessing Azure for read / write operations.

- `storage_account_key` **(required)** - The secret key that corresponds to
  the configured `storage_account`.

- `storage_container` **(required)** - The name of the storage container in
  which to store the backup archives.  This is essentially the analog to S3
  or GCS _buckets_.

- `prefix` - An optional prefix for backup archive paths.  This can be
  useful if you are sharing a bucket between multiple teams, or across
  two or more different environments, and want to be able to keep them
  separate for out-of-band retrieval.

  By default, no prefix will be used.


### OpenStack Swift

The `swift` plugin lets you store backup archives in an [OpenStack Swift
Blobstore][swift].

[swift]: https://docs.openstack.org/swift/latest/

Backup archives will be stored in a file name / path that encodes the date
and time of the backup operation, to make it easier to track down a specific
archive later:

    $prefix/YYYY/MM/DD/HHmmSS-$uuid

for example, given a `prefix` of "prod/backups", a backup might be stored
at:

    prod/backups/2018/07/12/134255-f3b564f2-ef62-4e38-9d94-ba17c37abf09


#### Configuration Options

- `auth_url` **(required)** - The URL of the OpenStack authentication API.

  V2 example: `https://identity.api.rackspacecloud.com/v2.0`.

  V3 example: `https://identity.api.rackspacecloud.com/v3.0`.

- `project_name` **(required for v2 auth only)** - The name of the OpenStack
  project/tenant that will own the blobstore data.

- `domain` **(required for v3 auth only)** - The name of the OpenStack domain
  that will own the blobstore data.

- `username` **(required)** - The username to authenticate to OpenStack.

- `password` **(required)** - The password for the given `username`.

- `container` **(required)** - The name of the blobstore container in which
  to store backup archives.  This is loosely analogous to an S3 bucket.

- `prefix` - An optional prefix for backup archive paths.  This can be
  useful if you are sharing a bucket between multiple teams, or across
  two or more different environments, and want to be able to keep them
  separate for out-of-band retrieval.

  By default, no prefix will be used.


### WebDAV Filesystem

The `webdav` plugin lets you store backup archives in any WebDAV server that
complies with [RFC 2518](https://tools.ietf.org/rfc/rfc2518.txt).  This
includes [Apache][apache-webdav] and [Nginx][nginx-webdav].

[apache-webdav]: https://httpd.apache.org/docs/2.4/mod/mod_dav.html
[nginx-webdav]:  http://nginx.org/en/docs/http/ngx_http_dav_module.html

#### Configuration Options

- `url` **(required)** - The full HTTP(S) URL of the WebDAV server.  This
  might include a path if you aren't storing files in the top of the server
  filesystem hierarchy.

- `username` - A username to use for HTTP Basic Authentication against the
  WebDAV server.  If not specified, no authentication is performed (which
  may not work).

- `password` - The password for the given `username`.

- `skip_ssl_validation` - Whether to verify the X.509 certificate of the
  WebDAV server, or not.  If your WebDAV host is using an untrusted,
  expired, or self-signed certificate, you can set this option to true to
  bypass verification failure.  **_This is not recommended_** for production
  use.


[shield-mongo]:    https://github.com/shieldproject/shield-addon-mongodb-boshrelease
[shield-mysql]:    https://github.com/shieldproject/shield-addon-mysql-boshrelease
[shield-postgres]: https://github.com/shieldproject/shield-addon-postgres-boshrelease
[shield-bbr]:      https://github.com/shieldproject/shield-addon-bbr-boshrelease
