# S.H.I.E.L.D. Backup Solution


## Project Goal
The goal of this project is to build a standalone system that can perform backup and restore functions for a wide variety of pluggable data systems (like Redis, PostgreSQL, MySQL, RabbitMQ, etc.), storing backup data in pluggable storage solutions (i.e. local files, S3 blobstore, etc.).

The system should enable self-service for end users to perform ad hoc backup / restore operations, review backup schedules, retention policies and backup job runs, etc.

Engineers should be able to integrate support for new data systems and storage solutions without having to modify core code.

## Architecture

![Architecture Image](https://raw.githubusercontent.com/starkandwayne/shield/master/docs/images/image00.png)

## Target Plugins

The system interfaces with data systems that hold the data to back up via Target Plugins.  These plugins are bits of code that are compiled and linked into the Core Daemon, and implement a standard interface for the following operations:

#### backup
Retrieves data from the data system (via native means like `pg_dump` or the Redis `SAVE` command) and sends it to an Storage Plugin.

#### restore
Retrieves the data from an Storage Plugin and overwrites the data in the data system accordingly, using native means like `pg_restore’

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
  plugin    TEXT,  -- short name of the target plugin, like 'postgres'
  endpoint  TEXT,  -- opaque blob used by target plugin to connect to
                   --   the remote data system.  Could be JSON, YAML, etc.
);
```

STORES stores the destination of backup data, i.e. an S3 bucket, local file system directory, etc.  Each record identifies a destination, the method by which to store and retrieve backup data to/from it (`plugin') and specific connection information required (`endpoint')

```sql
CREATE TABLE stores (
  uuid    UUID PRIMARY KEY,
  name    TEXT,  -- a human-friendly name for this store
  summary   TEXT,  -- annotation for operator use, to describe the store
  plugin  TEST,  -- short name of the storage plugin, like 's3' or 'fs'
  endpoint  TEXT,  -- opaque blob used by storage plugin to connect to
                   -- the storage backend.  Could be JSON, YAML, etc.
);
```

SCHEDULES contains the timing information that informs the core daemon when it should run which backup jobs (or JOBS, see later).

```sql
CREATE TABLE schedules (
  uuid    UUID PRIMARY KEY,
  name    TEXT, -- a human-friendly name for this schedule
  summary   TEXT, -- annotation for operator use, to describe schedule
  timespec  TEXT, -- code in a DSL for specifying when to run backups,
                  --   i.e. 'sundays 8am' or 'daily 1am'
                  --   (note: may want to eval use of cron here)
);
```

RETENTION policies govern how long data is kept.  For now, this is just a simple expiration time, with `name' and `summary' fields for annotation.

All backups taken MUST have a retention policy; no backups are kept indefinitely.

```sql
CREATE TABLE retention (
  uuid     UUID PRIMARY KEY,
  name     TEXT,    -- a human-friendly name for this retention policy
  summary  TEXT,    -- annotation for operator use, to describe policy
  expiry   INTEGER, -- how long (in seconds) before a given backup expires
);
```

JOBS keeps track of desired backup behavior, by marrying a target (the data to backup) with a store (where to send that data), according to a schedule (when to do the backups) and a retention policy (how long to keep the data for).

JOBS can be annotated by operators to provide context and justification for each job.  For example, tickets can be called out in the `notes' field to direct people to more information about when the backup job was requested, and why.

```sql
CREATE TABLE jobs (
  uuid          UUID PRIMARY KEY,
  target_uuid     UUID,    -- the target
  store_uuid  UUID,    -- the store
  schedule_uuid   UUID,    -- what schedule to use
  retention_uuid  UUID,    -- what retention policy to use
  paused        BOOLEAN, -- if true, this job is not run when scheduled.
  name          TEXT,    -- a human-friendly name for this schedule
  summary         TEXT,    -- annotation for operator use, to describe
                           --   the purpose of the job (‘weekly orders db’)
);
```

ARCHIVES records all archives as they are created, and keeps track of where the data came from, where it went, when the backed-up data expires, etc.

ARCHIVES can be annotated by operators, so that they can keep track of specifically important backups, like dumps of databases taken before potentially risky changes are attempted.

```sql
CREATE TABLE archives (
  uuid         UUID PRIMARY KEY,
  target_uuid  UUID, -- the target (from jobs)
  store_uuid   UUID, -- the store (from jobs)
  store_key    TEXT, -- opaque data returned from the storage plugin,
                     --   for use in restore ops / download / etc.
  taken_at     timestamp without time zone,
  expires_at   timestamp without time zone, -- based on retention policy
  notes        TEXT, -- annotation for operator use, to describe this
                     --   specific backup, i.e. 'before change #422 backup'
                     --   (mostly, this will be empty)
);
```

TASKS keep track of non-custodial jobs being performed by the system.  This includes scheduled backups, ad-hoc backups, data restoration and downloads, etc.

The core daemon interprets the `op' field, and calls on the appropriate plugins, based on the associated JOB or ARCHIVE entry. Additional arguments will be passed via the `args' field, which should be JSON.

Each TASK should be associated with either a JOB or an ARCHIVE.

Here are the defined operations:

|||
|---------------------------------------------------------------------------------------------------------------|
| backup | Perform a backup of the associated JOB. The target and store are pulled directly from the JOB entry. <br>Note: the `backup` operation is used for both ad hoc and scheduled backups. |
| restore | Perform a restore of the associated ARCHIVE.  The storage channel is pulled directly from the ARCHIVE. The target can be specified in the `args` JSON.  If it is not, the values from the ARCHIVE will be used.  This allows restores to go to a different host (for migration / scale-out purposes). |

```sql
CREATE TYPE status AS ENUM ('pending', 'running', 'canceled', 'done');
CREATE TABLE tasks (
  uuid      UUID PRIMARY KEY,
  owner     TEXT, -- who owns / started this task?
  op        TEXT, -- name of the operation to run, i.e. 'backup' or 'restore'
  args      TEXT, -- a JSON blob of arguments for the operation.

  JOB_uuid   UUID,
  archive_uuid  UUID,

  status      status, -- current status of the task
  started_at  timestamp without time zone,
  stopped_at  timestamp without time zone,

  log       TEXT, -- log of task activity
  debug     TEXT, -- more verbose logs, for troubleshooting ex post facto.
);
```

## HTTP API

### Schedules API

Purpose: allows the Web UI and CLI to find out what schedules are defined, and provides CRUD operations for schedule management.  Allowing queries to filter to unused=t or unused=f enables the frontends to show schedules that can be deleted safely.

| | | |
|----|----|----|
| GET | /v1/schedules | ?unused=[tf] |
| POST | /v1/schedules | |
| DELETE | /v1/schedule/:uuid | |
| PUT | /v1/schedule/:uuid | |


### Retention Policies API

Purpose: allows the Web UI and CLI to find out what retention policies are defined, and provides CRUD operations for policy management.  Allowing queries to filter to unused=t or unused=f enables the frontends to show retention policies that can be deleted safely.

| | | |
|----|----|----|
| GET | /v1/retention/policies | ?unused=[tf] |
| POST | /v1/retention/policies | |
| DELETE | /v1/retention/policy/:uuid | |
| PUT | /v1/retention/policy/:uuid | |


### Targets API

Purpose: allows the Web UI and CLI to review what targets have been defined, and allows updates to existing targets (to change endpoints or plugins, for example) and remove unused targets (i.e. retired / decommissioned services).

| | | |
|----|----|----|
| GET | /v1/targets | ?plugin=:name <br> ?unused=[tf] |
| POST | /v1/targets | |
| DELETE | /v1/target/:uuid | |
| PUT | /v1/target/:uuid | |


### Stores API

Purpose: allows operators (via the Web UI and CLI components) to view what storage systems are available for configuring backups, provision new ones, update existing ones and delete unused ones.

| | | |
|----|----|----|
| GET | /v1/stores | ?plugin=:name <br>?unused=[tf] |
| POST | /v1/stores | |
| DELETE | /v1/store/:uuid | |
| PUT | /v1/store/:uuid | |


### Jobs API
Purpose: allows end-users and operators to see what jobs have been configured, and the details of those configurations.  The filtering on the main listing / search endpoint (/v1/jobs) allows the frontends to show only jobs for specific schedules (what weekly backups are we running?), retention policies (what backups are we keeping for 90d or more?), and specific targets / stores.

| | | |
|----|----|----|
| GET | /v1/jobs | ?target=:uuid<br>?store=:uuid<br>?schedule=:uuid<br>?retention=:uuid<br>?paused=[tf] |
| POST | /v1/jobs | |
| DELETE | /v1/job/:uuid | |
| PUT | /v1/job/:uuid | |
| POST | /v1/job/:uuid/pause | |
| POST | /v1/job/:uuid/unpause | |


### Archive API

Purpose: allows end-users and operators to see what backups have been performed, optionally filtering them to specific targets (just the Cloud Foundry postgres database please), stores (what’s in S3?) and time windows (only show me backups before that data corruption incident).  It also facilitates restoration of data, and purging of backups ahead of schedule.

Note: the PUT /v1/archive/:uuid endpoint is only able to update the annotations (name and summary) for an archive.

| | | |
|----|----|----|
| GET | /v1/archives | ?target=:uuid <br>?store=:uuid <br>?after=:timespec <br>?before=:timespec |
| GET | /v1/archive/:uuid | |
| POST | /v1/archive/:uuid/restore | { target: $target_uuid } |
| DELETE | /v1/archive/:uuid | |
| PUT | /v1/archive/:uuid | |


### Tasks API
Purpose: allows the Web UI and the CLI to show running tasks, query a specific task, submit new tasks, cancel tasks, etc.

| | | |
|----|----|----|
| GET | /v1/tasks | ?status=:status <br>?debug |
| POST | /v1/tasks | |
| PUT | /v1/task/:uuid | |
| DELETE | /v1/task/:uuid | |

## Plugin Calling Protocol

Store and Target Plugins are implemented as external programs,
either scripts or compiled binaries, that follow the Plugin
Calling Protocol, which stipulates how file descriptors are to be
used, and what arguments are going to be passed to the external
program to perform what functions.

```bash
$ redis-plugin info
{"name":"My Redis Plugin","author":"Joe Random Hacker","version":"1.0.0","features":{"target":"yes","store":"no"}}

$ s3-plugin info
{"name":"My S3 Storage Plugin","author":"Joe Random Hacker","version":"2.1.4","features":{"target":"no","store":"yes"}}

$ redis-plugin backup -c $REDIS_ENDPOINT | s3-plugin store -c $S3_ENDPOINT
{"key":"BA670360-DE9D-46D0-AEAB-55E72BD416C4"}

$ s3-plugin retrieve -c $S3_ENDPOINT -k BA670360-DE9D-46D0-AEAB-55E72BD416C4 | redis-plugin restore -c $REDIS_ENDPOINT
```

Each plugin program must implement the following actions, which will be passed as the first argument:

- *info* Dump a JSON-encoded map containing the following keys, to standard output:

  1. *name* - The name of the plugin (human-readable)
  2. *author* - The name of the person or team who maintains the plugin.
     May include email, at author discretion.
  3. *version* - The version of the plugin
  4. *features* - A map of the features of this plugin.  Currently supports two boolean keys
     ("yes" for true, "no" for false, both lower case) named "target" and "store", that indicate
     whether or not the plugin can support target and/or store operations.

  Other keys are allowed, but ignored, and all keys are reserved for future expansion.  Keys starting
  with an underscore ('\_') will never be used by shield, and is free for your own use.

  Always exits 0 to signify success.  Exits non-zero to signify an error, and prints
  diagnostic information to standard error.

- *backup* Stream a backup blob of arbitrary binary data (per plugin semantics) to standard
  output, based on the endpoint given via the `-c` argument.  For example, a database target
  plugin may require the DSN and username/password in a JSON structure to be given via `-c`,
  and will run a platform-specific backup tool, hooking its output to standard output (like
  pgdump or mysqldump).

  Error messages and diagnostics should be printed to standard error.

  Exits 0 on success, or non-zero on failure.

- *restore* Read a backup blob of arbitrary binary data (per plugin semantics) from standard
  input, and perform a restore based on the endpoint given via the `-c` argument.

  Error messages and diagnostics should be printed to standard error.

  Exits 0 on success, or non-zero on failure.

- *store* Read a backup blob of arbitrary binary data from standard input, and store it in
  the remote storage system, based on the endpoint given via the `-c` argument.  For example,
  an S3 plugin might require keys and a bucket name to perform storage operations.

  Error messages and diagnostics should be printed to standard error.

  Exits 0 on success, or non-zero on failure.

  On success, write the JSON representation of a map containing a summary of the stored object,
  including the following keys:

  - *key* - An opaque identifier that means something to the plugin for purposes of restore.
    This will be logged in the database by shield.

- *retrieve* Stream a backup blob of arbitrary binary data to standard output, based on the
  endpoint configuration given as the `-c` argument, and a key, as given by the `-k` argument.
  (This will be the key that was returned from the *store* operation)

  Error messages and diagnostics should be printed to standard error.

  Exits 0 on success, or non-zero on failure.

## CLI Usage Examples

This section is exploratory.

```bash
# schedule management
$ bkp list schedules [--[un]used]
$ bkp show schedule $UUID
$ bkp delete schedule $UUID
$ bkp update schedule $UUID

# retention policies
$ bkp list retention policies [--[un]used]
$ bkp show retention policy $UUID
$ bkp delete retention policy $UUID
$ bkp update retention policy $UUID

# “managing” plugins
$ bkp list plugins
$ bkp show plugin $NAME

# targets
$ bkp list targets [--[un]used] [--plugin $NAME]
$ bkp show target $UUID
$ bkp edit target $UUID
$ bkp delete target $UUID

# stores
$ bkp list stores [--[un]used] [--plugin $NAME]
$ bkp show store $UUID
$ bkp edit store $UUID
$ bkp delete store $UUID

# jobs
$ bkp list jobs [--[un]paused] [--target $UUID] [--store $UUID]
                 [--schedule $UUID] [--retention-policy $UUID]
$ bkp show job $UUID
$ bkp pause job $UUID
$ bkp unpause job $UUID
$ bkp paused job $UUID
$ bkp run job $UUID
$ bkp edit job $UUID
$ bkp delete job $UUID

# archives
$ bkp list archives [--[un]paused] [--target $UUID] [--store $UUID]
                    [--after $TIMESPEC] [--before $TIMESPEC]
$ bkp show archive $UUID
$ bkp edit archive $UUID
$ bkp delete archive $UUID
$ bkp restore archive $UUID [--to $TARGET_UUID]

# task management
$ bkp list tasks [--all]
$ bkp show task $UUID [--debug]
$ bkp cancel task $UUID
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
    * bkp * job
    * bkp * backup
    * bkp * task

This will let us test flush out any inconsistencies in the architecture, and find any problematic aspects of the problem domain not presently considered.

### Stage 2 Proof-of-Concept
Next, we extend the proof-of-concept implementation to test out the Agent Target Plugin design, using Redis as the data system.  This entails the following:

* Implement the Agent Daemon (in general)
* Extend the Agent Daemon to handle Redis’ BGSAVE command
* Implement the Agent Target Plugin

