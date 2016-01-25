# S.H.I.E.L.D. Backup Solution
[![Build Status](https://travis-ci.org/starkandwayne/shield.svg)](https://travis-ci.org/starkandwayne/shield)

## Project Goal
The goal of this project is to build a standalone system that can perform backup and restore functions for a wide variety of pluggable data systems (like Redis, PostgreSQL, MySQL, RabbitMQ, etc.), storing backup data in pluggable storage solutions (i.e. local files, S3 blobstore, etc.).

The system should enable self-service for end users to perform ad hoc backup / restore operations, review backup schedules, retention policies and backup job runs, etc.

Engineers should be able to integrate support for new data systems and storage solutions without having to modify core code.

## Architecture

![Architecture Image](https://raw.githubusercontent.com/starkandwayne/shield/master/docs/images/image00.png)

## Task Lifecycle

![Task Lifecyle Image](https://raw.githubusercontent.com/starkandwayne/shield/master/docs/images/task-lifecycle.png)

## Target Plugins

The system interfaces with data systems that hold the data to back up via Target Plugins.  These plugins are bits of code that are compiled and linked into the Core Daemon, and implement a standard interface for the following operations:

#### backup
Retrieves data from the data system (via native means like `pg_dump` or the Redis `SAVE` command) and sends it to an Storage Plugin.

#### restore
Retrieves the data from an Storage Plugin and overwrites the data in the data system accordingly, using native means like `pg_restore`.

For data systems that permit full backups across a network (as most RDBMS do), nothing more is needed.  Some data systems, however, make assumptions about the environment in which they operate.  Redis, for example, always dumps its backups to local disk.  To support these data systems, we can implement the Agent Target Plugin, and a corresponding Agent Daemon that will run on the target system.  The Agent Daemon will be responsible for implementing the backup / restore options, and the Agent Target Plugin will forward the requests to it, and relay responses back to the caller.

## Storage Plugins

The system interfaces with storage systems for uploading and retrieving backed up data files.  These plugins are bits of code that are compiled and linked into the Core Daemon, and implements a standard interface for the following operations:

#### store
Store a single data blob (usually a file) in the remote storage system.  Returns a key that can be used for later retrieval.

#### retrieve
Given a key returned from the store operation, retrieve the data blob.

#### purge
Given a key returned from the store operation, delete the stored data.


## Core Daemon

The Core Daemon is the coordinating component that handles:

#### Metadata Management
What targets and stores exist, what schedules and retention policies are defined, what jobs are specified, what backups have taken place, and what tasks are in-flight.
#### Scheduling Backups
Kicks off backup tasks (owned by SYSTEM) for all jobs per their configured schedule.
#### Expiring Backups
Finds all expired entries in the archives and purges them from the remote storage system.
#### Ad hoc Backups
Kicks off backup tasks (owned by users) per end-user or operator request (via the HTTP API, detailed later.)
#### Restores
Handles retrieval of stored backup data and replay / restoration of that data to a given target.
#### Monitoring
Exposes metrics and statistics about backup jobs, allows searching of archives to ensure that backups are completing successfully, etc.

## HTTP API

The HTTP API is a component of the Core Daemon that exposes management interfaces via REST endpoints.  It underlies the Web UI and CLI components (described later).

Detailed documentation on the HTTP API can be read in the [docs/api/http.md](https://github.com/starkandwayne/shield/blob/master/docs/api/http.md) file.

## Catalog Database

A dedicated data store that keeps track of schedules, retention policies, backup configurations, targets and stores, and running tasks.  This database is private to the Core Daemon; there should be no need to query it directly, outside of maintenance tasks.
Web UI and the CLI

The Web UI provides a rich user interface for operators and end-users to view configuration (schedules, policies, jobs, etc.) review archives, and monitor tasks in-progress.  It also provides self-service functionality by allowing users to request ad hoc backup and restore operations.

The Web UI relies exclusively on the HTTP API.

The CLI provides similar functionality, in a scriptable, command-line interface.  It also relies exclusively on the HTTP API.
Catalog Database Schema Definition

TARGETS stores the information about the remote data systems that should be backed up.  Each record identifies the method by which the target is backed up (`plugin`) and specific connection information required (`endpoint`)

```sql
CREATE TABLE targets (
  uuid      UUID PRIMARY KEY,
  name      TEXT,  -- a human-friendly name for this target
  summary   TEXT,  -- annotation for operator use, to describe the target
                   --   i.e.: "Production PostgreSQL database"
  plugin    TEXT NOT NULL,  -- short name of the target plugin, like 'postgres'
  endpoint  TEXT NOT NULL,  -- opaque blob used by target plugin to connect to
                            --   the remote data system.  Could be JSON, YAML, etc.
  agent     TEXT NOT NULL,  -- IP address and port (in ip:port format) of the
                            -- Shield agent that can backup/restore this target
);
```

STORES stores the destination of backup data, i.e. an S3 bucket, local file system directory, etc.  Each record identifies a destination, the method by which to store and retrieve backup data to/from it ('plugin') and specific connection information required ('endpoint')

```sql
CREATE TABLE stores (
  uuid      UUID PRIMARY KEY,
  name      TEXT,  -- a human-friendly name for this store
  summary   TEXT,  -- annotation for operator use, to describe the store
  plugin    TEXT NOT NULL,  -- short name of the storage plugin, like 's3' or 'fs'
  endpoint  TEXT NOT NULL,  -- opaque blob used by storage plugin to connect to
                            -- the storage backend.  Could be JSON, YAML, etc.
);
```

SCHEDULES contains the timing information that informs the core daemon when it should run which backup jobs (or JOBS, see later).

```sql
CREATE TABLE schedules (
  uuid      UUID PRIMARY KEY,
  name      TEXT, -- a human-friendly name for this schedule
  summary   TEXT, -- annotation for operator use, to describe schedule
  timespec  TEXT NOT NULL, -- code in a DSL for specifying when to run backups,
                           --   i.e. 'sundays 8am' or 'daily 1am'
                           --   (note: may want to eval use of cron here)
);
```

RETENTION policies govern how long data is kept.  For now, this is just a simple expiration time, with 'name' and 'summary' fields for annotation.

All backups taken MUST have a retention policy; no backups are kept indefinitely.

```sql
CREATE TABLE retention (
  uuid     UUID PRIMARY KEY,
  name     TEXT, -- a human-friendly name for this retention policy
  summary  TEXT, -- annotation for operator use, to describe policy
  expiry   INTEGER NOT NULL, -- how long (in seconds) before a given backup expires
);
```

JOBS keeps track of desired backup behavior, by marrying a target (the data to backup) with a store (where to send that data), according to a schedule (when to do the backups) and a retention policy (how long to keep the data for).

JOBS can be annotated by operators to provide context and justification for each job.  For example, tickets can be called out in the `notes` field to direct people to more information about when the backup job was requested, and why.

```sql
CREATE TABLE jobs (
  uuid            UUID PRIMARY KEY,
  target_uuid     UUID NOT NULL, -- the target
  store_uuid      UUID NOT NULL, -- the store
  schedule_uuid   UUID NOT NULL, -- what schedule to use
  retention_uuid  UUID NOT NULL, -- what retention policy to use
  priority        INTEGER DEFAULT 50, -- priority, scale from 0 to 100 (0 = highest)
  paused          BOOLEAN, -- if true, this job is not run when scheduled.
  name            TEXT,    -- a human-friendly name for this schedule
  summary         TEXT,    -- annotation for operator use, to describe
                           --   the purpose of the job ('weekly orders db')
);
```

ARCHIVES records all archives as they are created, and keeps track of where the data came from, where it went, when the backed-up data expires, etc.

ARCHIVES can be annotated by operators, so that they can keep track of specifically important backups, like dumps of databases taken before potentially risky changes are attempted.

```sql
CREATE TABLE archives (
  uuid         UUID PRIMARY KEY,
  target_uuid  UUID NOT NULL, -- the target (from jobs)
  store_uuid   UUID NOT NULL, -- the store (from jobs)
  store_key    TEXT NOT NULL, -- opaque data returned from the storage plugin,
                              --   for use in restore ops / download / etc.
  taken_at     INTEGER NOT NULL,
  expires_at   INTEGER NOT NULL, -- based on retention policy
  notes        TEXT DEFAULT '', -- annotation for operator use, to describe this
                                --   specific backup, i.e. 'before change #422 backup'
                                --   (mostly, this will be empty)
);
```

TASKS keep track of non-custodial jobs being performed by the system.  This includes scheduled backups, ad-hoc backups, data restoration and downloads, etc.

The core daemon interprets the 'op' field, and calls on the appropriate plugins, based on the associated JOB or ARCHIVE / TARGET entry.

Each TASK should be associated with either a JOB or an ARCHIVE.

Here are the defined operations:

| Operation | Description |
| :-------- | :---------- |
| backup | Perform a backup of the associated JOB. The target and store are pulled directly from the JOB entry. <br>Note: the `backup` operation is used for both ad hoc and scheduled backups. |
| restore | Perform a restore of the associated ARCHIVE.  The storage channel is pulled directly from the ARCHIVE. The target can be specified explicitly.  If it is not, the values from the ARCHIVE will be used.  This allows restores to go to a different host (for migration / scale-out purposes). |

```sql
CREATE TYPE status AS ENUM ('pending', 'running', 'canceled', 'failed', 'done');
CREATE TABLE tasks (
  uuid      UUID PRIMARY KEY,
  owner     TEXT, -- who owns / started this task?
  op        TEXT NOT NULL, -- name of the operation to run, i.e. 'backup' or 'restore'

  job_uuid      UUID,
  archive_uuid  UUID,
  target_uuid   UUID,

  status       status, -- current status of the task
  requested_at INTEGER NOT NULL, -- when the task was _created_
  started_at   INTEGER, -- when the task actually started
  stopped_at   INTEGER, -- when the task completed (or was cancelled)

  log       TEXT -- log of task activity
);
```

## Plugin Calling Protocol

Store and Target Plugins are implemented as external programs,
either scripts or compiled binaries, that follow the Plugin
Calling Protocol, which stipulates how file descriptors are to be
used, and what arguments are going to be passed to the external
program to perform what functions.

```bash
$ redis-plugin info
{
  "name": "My Redis Plugin",
  "author": "Joe Random Hacker",
  "version": "1.0.0",
  "features": {
    "target": "yes",
    "store": "no"
  }
}

$ s3-plugin info
{
  "name": "My S3 Storage Plugin",
  "author": "Joe Random Hacker",
  "version": "2.1.4",
  "features": {
    "target": "no",
    "store": "yes"
  }
}

$ redis-plugin backup --endpoint '{"username":"redis","password":"secret"}' | s3-plugin store --endpoint '{"bucket":"test","key":"AKI123098123091"}'
{
  "key": "BA670360-DE9D-46D0-AEAB-55E72BD416C4"
}

$ s3-plugin retrieve --key decaf-bad --endpoint '{"bucket":"test","key":"AKI123098123091"}' | redis-plugin restore --endpoint '{"username":"redis","password":"secret"}'
```

Each plugin program must implement the following actions, which will be passed as the first argument:

- **info** - Dump a JSON-encoded map containing the following keys, to standard output:

  1. `name` - The name of the plugin (human-readable)
  2. `author` - The name of the person or team who maintains the plugin.
     May include email, at author discretion.
  3. `version` - The version of the plugin
  4. `features` - A map of the features of this plugin.  Currently supports two boolean keys
     ("yes" for true, "no" for false, both lower case) named "target" and "store", that indicate
     whether or not the plugin can support target and/or store operations.

  Other keys are allowed, but ignored, and all keys are reserved for future expansion.  Keys starting
  with an underscore ('\_') will never be used by shield, and is free for your own use.

  Always exits 0 to signify success.  Exits non-zero to signify an error, and prints
  diagnostic information to standard error.

- **backup** - Stream a backup blob of arbitrary binary data (per
  plugin semantics) to standard output, based on the endpoint
  given via the `--endpoint` command line argument.
  For example, a database target plugin may require the DSN and
  username/password in a JSON structure, and will run a
  platform-specific backup tool, hooking its output to standard
  output (like pgdump or mysqldump).

  Error messages and diagnostics should be printed to standard error.

  Exits 0 on success, or non-zero on failure.

- **restore** - Read a backup blob of arbitrary binary data (per
  plugin semantics) from standard input, and perform a restore
  based on the endpoint given via the `--endpoint` command line argument.

  Error messages and diagnostics should be printed to standard error.

  Exits 0 on success, or non-zero on failure.

- **store** - Read a backup blob of arbitrary binary data from
  standard input, and store it in the remote storage system, based
  on the endpoint given via the `--endpoint` command line argument.
  For example, an S3 plugin might require keys and a bucket name to
  perform storage operations.

  Error messages and diagnostics should be printed to standard error.

  Exits 0 on success, or non-zero on failure.

  On success, write the JSON representation of a map containing a summary of the stored object,
  including the following keys:

  1. `key` - An opaque identifier that means something to the plugin for purposes of restore.
     This will be logged in the database by shield.

  Other keys are allowed, but ignored, and all keys are reserved for future expansion.  Keys starting
  with an underscore ('\_') will never be used by shield, and is free for your own use.

- **retrieve** Stream a backup blob of arbitrary binary data to
  standard output, based on the endpoint configuration given in
  the `--endpoint` command line argument, and a key, as
  given by the `--key` command line argument.  (This
  will be the key that was returned from the **store** operation)

  Error messages and diagnostics should be printed to standard error.

  Exits 0 on success, or non-zero on failure.

- **purge** Remove a backup blob of arbitrary data from the remote
  storage system, based on the endpoint configuration given in
  the `--endpoint` command line argument. The blob to be removed is
  identified via the `--key` command line argument.

  Error messages and diagnostics should be printed to standard error.

  Exits 0 on success, or non-zero on failure.

## Notes on Development

Setting the environment variable `SHIELD_MODE` to the value `DEV`
will cause all scheduling information to revert to "every minute"
regardless of the actual schedule.  This is to assist developers.

## The Makefile

The Makefile is used to assist with development. The available targets are:
* `test` | `tests` : runs all the tests with no additional parameters
 * `coverage` : runs tests with coverage information
 * `report` : makes report in (temporary) HTML page for a particular package, e.g. `db`. See examples.
* `race` : runs `ginkgo -race *` to test for race conditions
* `plugin` | `plugins` : builds all the plugin binaries
* `shield` : builds the `shieldd`, `shield-schema`, `shield-agent`, and `shield` (CLI) binaries
* `all` : runs all the tests (except the race test) and builds all the binaries.
* `fixme` | `fixmes` : finds all FIXMEs in the project

`all` is also the default behavior, so running `make` with no targets is the same as `make all`.

Examples:

```
$ make shield
go build ./cmd/shieldd
go build ./cmd/shield-agent
go build ./cmd/shield-schema
go build ./cmd/shield

$ make tests
ginkgo *
[1450032890] Agent Test Suite - 39/39 specs •••••••••••••••••••••••••••••••••••••• SUCCESS! 387.609253ms PASS
[1450032890] API Client Library Test Suite - 3/3 specs ••• SUCCESS! 185.602µs PASS
[1450032890] Database Layer Test Suite - 21/21 specs ••••••••••••••••••••• SUCCESS! 15.888175ms PASS
[1450032890] Plugin Framework Test Suite - 45/45 specs ••••••••••••••••••••••••••••••••••••••••••••• SUCCESS! 20.695859ms PASS
[1450032890] Supervisor Test Suite - 139/139 specs ••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••• SUCCESS! 155.843391ms PASS
[1450032890] Timespec Test Suite - 37/37 specs ••••••••••••••••••••••••••••••••••••• SUCCESS! 26.84143ms PASS

Ginkgo ran 6 suites in 4.001600857s
Test Suite Passed
go vet ./...

$ make report FOR=db
go tool cover -html=coverage/db.cov
```

## CLI Usage Examples

This section is exploratory.

```
# API
$ export SHIELD_API="${SHIELD_API_IP}:${SHIELD_API_PORT}"; shield status

# info
$ shield help
$ shield status

# targets
$ shield list targets [--[un]used] [--plugin $NAME]
$ shield show target $UUID
$ shield create target
$ shield edit target $UUID
$ shield delete target $UUID

# schedule management
$ shield list schedules [--[un]used]
$ shield show schedule $UUID
$ shield create schedule
$ shield update schedule $UUID
$ shield delete schedule $UUID

# retention policies
$ shield list retention policies [--[un]used]
$ shield show retention policy $UUID
$ shield create retention policy
$ shield update retention policy $UUID
$ shield delete retention policy $UUID

# stores
$ shield list stores [--[un]used] [--plugin $NAME]
$ shield show store $UUID
$ shield create store
$ shield edit store $UUID
$ shield delete store $UUID

# jobs
$ shield list jobs [--[un]paused] [--target $UUID] [--store $UUID]
                [--schedule $UUID] [--retention-policy $UUID]
$ shield show job $UUID
$ shield create job
$ shield edit job $UUID
$ shield delete job $UUID
$ shield pause job $UUID
$ shield unpause job $UUID
$ shield paused job $UUID
$ shield run job $UUID

# archives
$ shield list archives [--target $UUID] [--store $UUID]
                    [--after YYYYMMDD] [--before YYYYMMDD]
$ shield show archive $UUID
$ shield delete archive $UUID
$ shield restore archive $UUID [--to $TARGET_UUID]

# task management
$ shield list tasks [--all]
$ shield show task $UUID
$ shield cancel task $UUID
```

## Proof of Concept (Where Do We Go From Here?)

### Research

We need to identify all of the data systems we wish to support with this system.  For each system, we need to identify any problematic systems that will not fit into one of the two collection / restore models designed:

* Direct over-the-network backup/restore a la pg_dump / pg_restore
* Instrumentation of local backup/restore + file shipping via Agent Daemon / Plugin

### Stage 1 Proof-of-Concept

To get this project off the ground, I think we need to do some research and experimental implementation into the following areas:

* Implement the postgres target plugin using pg_dump / pg_restore tools
* Implement the fs storage plugin to store blobs in the local file system
* Implement the Core Daemon with limited functionality:
    * Task execution
    * backup operation
    * restore operation
* Implement the HTTP API with limited functionality:
    * /v1/jobs/*
    * /v1/archive/*
* Implement the CLI with limited functionality:
    * shield * job
    * shield * backup
    * shield * task

This will let us test flush out any inconsistencies in the architecture, and find any problematic aspects of the problem domain not presently considered.

### Stage 2 Proof-of-Concept
Next, we extend the proof-of-concept implementation to test out the Agent Target Plugin design, using Redis as the data system.  This entails the following:

* Implement the Agent Daemon (in general)
* Extend the Agent Daemon to handle Redis’ BGSAVE command
* Implement the Agent Target Plugin
