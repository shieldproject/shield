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

STORES stores the destination of backup data, i.e. an S3 bucket, local file system directory, etc.  Each record identifies a destination, the method by which to store and retrieve backup data to/from it ('plugin') and specific connection information required ('endpoint')

```sql
CREATE TABLE stores (
  uuid      UUID PRIMARY KEY,
  name      TEXT,  -- a human-friendly name for this store
  summary   TEXT,  -- annotation for operator use, to describe the store
  plugin    TEXT,  -- short name of the storage plugin, like 's3' or 'fs'
  endpoint  TEXT,  -- opaque blob used by storage plugin to connect to
                   -- the storage backend.  Could be JSON, YAML, etc.
);
```

SCHEDULES contains the timing information that informs the core daemon when it should run which backup jobs (or JOBS, see later).

```sql
CREATE TABLE schedules (
  uuid      UUID PRIMARY KEY,
  name      TEXT, -- a human-friendly name for this schedule
  summary   TEXT, -- annotation for operator use, to describe schedule
  timespec  TEXT, -- code in a DSL for specifying when to run backups,
                  --   i.e. 'sundays 8am' or 'daily 1am'
                  --   (note: may want to eval use of cron here)
);
```

RETENTION policies govern how long data is kept.  For now, this is just a simple expiration time, with 'name' and 'summary' fields for annotation.

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

JOBS can be annotated by operators to provide context and justification for each job.  For example, tickets can be called out in the `notes` field to direct people to more information about when the backup job was requested, and why.

```sql
CREATE TABLE jobs (
  uuid            UUID PRIMARY KEY,
  target_uuid     UUID,    -- the target
  store_uuid      UUID,    -- the store
  schedule_uuid   UUID,    -- what schedule to use
  retention_uuid  UUID,    -- what retention policy to use
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
  target_uuid  UUID, -- the target (from jobs)
  store_uuid   UUID, -- the store (from jobs)
  store_key    TEXT, -- opaque data returned from the storage plugin,
                     --   for use in restore ops / download / etc.
  taken_at     timestamp without time zone,
  expires_at   timestamp without time zone, -- based on retention policy
  notes        TEXT DEFAULT "", -- annotation for operator use, to describe this
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
  op        TEXT, -- name of the operation to run, i.e. 'backup' or 'restore'

  job_uuid      UUID,
  archive_uuid  UUID,
  target_uuid   UUID,

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

| Method | Path | Arguments | Request Body |
| :----- | :---- | :------- | :----------- |
| GET | /v1/schedules | ?unused=[tf] | - |
| POST | /v1/schedules | - | see below |
| DELETE | /v1/schedule/:uuid | - | - |
| PUT | /v1/schedule/:uuid | - | see below |
|
#### GET /v1/schedules

Response Body:

```json
[
  {
    "uuid"    : "36f50f26-b007-433a-a67a-bdffbd0746c8",
    "name"    : "Schedule Name",
    "summary" : "a short description",
    "when"    : "daily at 4am"
  },

  "..."
]
```

#### POST /v1/schedules

Request Body:

```json
{
  "name"    : "Schedule Name",
  "summary" : "a short description",
  "when"    : "daily at 4am"
}
```

| Field | Required? | Meaning |
| :---- | :-------: | :------ |
| name | Y | The name of the new schedule
| summary | N | A short summary of what the schedule is for, when it should be used
| when | Y | The schedule, in the Timespec Language

Response Body:

```json
{
  "ok"   : "created",
  "uuid" : "6b8398be-fdc0-424a-8532-e812e5dfc116"
}
```

| Field | Meaning |
| :---- | :------ |
| ok    | The new schedule was created
| uuid  | The UUID of the newly-created schedule

#### PUT /v1/schedule/:uuid

Request Body:

```json
{
  "name"    : "Schedule Name",
  "summary" : "a short description",
  "when"    : "daily at 4am"
}
```

| Field | Required? | Meaning |
| :---- | :-------: | :------ |
| name | Y | The name of the new schedule
| summary | Y | A short summary of what the schedule is for, when it should be used
| when | Y | The schedule, in the Timespec Language

**NOTE:** `summary` is required for update requests, whereas it is optional on creation.

Response Body:

```json
{
  "ok" : "updated"
}
```

| Field | Meaning |
| :---- | :------ |
| ok    | The schedule was updated



### Retention Policies API

Purpose: allows the Web UI and CLI to find out what retention policies are defined, and provides CRUD operations for policy management.  Allowing queries to filter to unused=t or unused=f enables the frontends to show retention policies that can be deleted safely.

| Method | Path | Arguments | Request Body |
| :----- | :---- | :------- | :----------- |
| GET | /v1/retention | ?unused=[tf] | - |
| POST | /v1/retention | - | see below |
| DELETE | /v1/retention/:uuid | - | - |
| PUT | /v1/retention/:uuid | - | see below |

#### GET /v1/retention

```json
[
  {
    "uuid"    : "c5aed303-a6fc-4b68-b0e9-81431cc07a4e",
    "name"    : "Retention Policy Name",
    "summary" : "a short description",
    "expires" : 86400
  },

  "..."
]
```

#### POST /v1/retention

Request Body:

```json
{
  "name"    : "Policy Name",
  "summary" : "a short description",
  "when"    : 86400
}
```

| Field | Required? | Meaning |
| :---- | :-------: | :------ |
| name | Y | The name of the new retention policy
| summary | N | A short summary of the new retention policy
| expires | Y | How long, in seconds, to keep archives made against this policy.  This value must be at least 3600 (1h)

Response Body:

```json
{
  "ok"   : "created",
  "uuid" : "6b8398be-fdc0-424a-8532-e812e5dfc116"
}
```

| Field | Meaning |
| :---- | :------ |
| ok    | The new retention policy was created
| uuid  | The UUID of the newly-created retention policy

#### PUT /v1/retention/:uuid

Request Body:

```json
{
  "name"    : "Policy Name",
  "summary" : "a short description",
  "when"    : 86400
}
```

| Field | Required? | Meaning |
| :---- | :-------: | :------ |
| name | Y | The name of the new retention policy
| summary | Y | A short summary of the new retention policy
| expires | Y | How long, in seconds, to keep archives made against this policy.  This value must be at least 3600 (1h)

**NOTE:** `summary` is required for update requests, whereas it is optional on creation.

Response Body:

```json
{
  "ok" : "updated"
}
```

| Field | Meaning |
| :---- | :------ |
| ok    | The retention policy was updated



### Targets API

Purpose: allows the Web UI and CLI to review what targets have been defined, and allows updates to existing targets (to change endpoints or plugins, for example) and remove unused targets (i.e. retired / decommissioned services).

| Method | Path | Arguments | Request Body |
| :----- | :---- | :------- | :----------- |
| GET | /v1/targets | ?plugin=:name <br> ?unused=[tf] | - |
| POST | /v1/targets | - | see below |
| DELETE | /v1/target/:uuid | - | - |
| PUT | /v1/target/:uuid | - | see below |

#### GET /v1/targets

```json
[
  {
    "uuid"     : "2f42d0b3-449a-4d0e-8576-a40cc552d7e5",
    "name"     : "Target Name",
    "summary"  : "a short description",
    "plugin"   : "plugin-name",
    "endpoint" : "{\"encoded\":\"json\"}"
  },

  "..."
]
```

#### POST /v1/targets

Request Body:

```json
{
  "name"     : "Target Name",
  "summary"  : "a short description",
  "plugin"   : "plugin-name",
  "endpoint" : "{\"encoded\":\"json\"}"
}
```

| Field | Required? | Meaning |
| :---- | :-------: | :------ |
| name | Y | The name of the new target
| summary | N | A short description of the target
| plugin | Y | The name of the plugin to use when backing up this target
| endpoint | Y | The endpoint configuration required to access this target's data

Response Body:

```json
{
  "ok"   : "created",
  "uuid" : "6b8398be-fdc0-424a-8532-e812e5dfc116"
}
```

| Field | Meaning |
| :---- | :------ |
| ok    | The new target was created
| uuid  | The UUID of the newly-created target

#### PUT /v1/target/:uuid

Request Body:

```json
{
  "name"     : "Target Name",
  "summary"  : "a short description",
  "plugin"   : "plugin-name",
  "endpoint" : "{\"encoded\":\"json\"}"
}
```

| Field | Required? | Meaning |
| :---- | :-------: | :------ |
| name | Y | The name of the new target
| summary | Y | A short description of the target
| plugin | Y | The name of the plugin to use when backing up this target
| endpoint | Y | The endpoint configuration required to access this target's data
|
**NOTE:** `summary` is required for update requests, whereas it is optional on creation.

Response Body:

```json
{
  "ok" : "updated"
}
```

| Field | Meaning |
| :---- | :------ |
| ok    | The target was updated



### Stores API

Purpose: allows operators (via the Web UI and CLI components) to view what storage systems are available for configuring backups, provision new ones, update existing ones and delete unused ones.

| Method | Path | Arguments | Request Body |
| :----- | :---- | :------- | :----------- |
| GET | /v1/stores | ?plugin=:name <br>?unused=[tf] | - |
| POST | /v1/stores | - | see below |
| DELETE | /v1/store/:uuid | - | - |
| PUT | /v1/store/:uuid | - | see below |

#### GET /v1/stores

```json
[
  {
    "uuid"     : "5bcde12a-8b3f-4663-bbe3-9fe0fd6a093d",
    "name"     : "Store Name",
    "summary"  : "a short description",
    "plugin"   : "plugin-name",
    "endpoint" : "{\"encoded\":\"json\"}"
  },

  "..."
]
```

#### POST /v1/stores

Request Body:

```json
{
  "name"     : "Store Name",
  "summary"  : "a short description",
  "plugin"   : "plugin-name",
  "endpoint" : "{\"encoded\":\"json\"}"
}
```

| Field | Required? | Meaning |
| :---- | :-------: | :------ |
| name | Y | The name of the new store
| summary | N | A short description of the store
| plugin | Y | The name of the plugin to use when backing up this store
| endpoint | Y | The endpoint configuration required to access this store's data

Response Body:

```json
{
  "ok"   : "created",
  "uuid" : "6b8398be-fdc0-424a-8532-e812e5dfc116"
}
```

| Field | Meaning |
| :---- | :------ |
| ok    | The new store was created
| uuid  | The UUID of the newly-created store

#### PUT /v1/store/:uuid

Request Body:

```json
{
  "name"     : "Store Name",
  "summary"  : "a short description",
  "plugin"   : "plugin-name",
  "endpoint" : "{\"encoded\":\"json\"}"
}
```

| Field | Required? | Meaning |
| :---- | :-------: | :------ |
| name | Y | The name of the new store
| summary | Y | A short description of the store
| plugin | Y | The name of the plugin to use when backing up this store
| endpoint | Y | The endpoint configuration required to access this store's data
|
**NOTE:** `summary` is required for update requests, whereas it is optional on creation.

Response Body:

```json
{
  "ok" : "updated"
}
```

| Field | Meaning |
| :---- | :------ |
| ok    | The store was updated



### Jobs API
Purpose: allows end-users and operators to see what jobs have been configured, and the details of those configurations.  The filtering on the main listing / search endpoint (/v1/jobs) allows the frontends to show only jobs for specific schedules (what weekly backups are we running?), retention policies (what backups are we keeping for 90d or more?), and specific targets / stores.

| Method | Path | Arguments | Request Body |
| :----- | :---- | :------- | :----------- |
| GET | /v1/jobs | ?target=:uuid<br>?store=:uuid<br>?schedule=:uuid<br>?retention=:uuid<br>?paused=[tf] | - |
| POST | /v1/jobs | - | see below |
| DELETE | /v1/job/:uuid | - | - |
| PUT | /v1/job/:uuid | - | see below |
| POST | /v1/job/:uuid/pause | - | - |
| POST | /v1/job/:uuid/unpause | - | - |
| POST | /v1/job/:uuid/run | - | see below |

#### GET /v1/jobs

```json
[
  {
    "uuid"            : "af0b40b2-8f7b-46e4-b425-9730c677e625",
    "name"            : "A Backup Job",
    "summary"         : "a short description",

    "retention_name"  : "100d Retention Policy",
    "retention_uuid"  : "7eb2131c-c2ad-40b1-916f-7e162be89465",
    "expiry"          : 8640000,

    "schedule_name"   : "Daily Backups Schedule",
    "schedule_uuid"   : "e390934b-fc43-4343-a51b-22bd69a8894f",
    "schedule"        : "daily at 4am",

    "paused"          : false,

    "store_plugin"    : "store-plugin",
    "store_endpoint"  : "{\"encoded\":\"json\"}",

    "target_plugin"   : "target-plugin",
    "target_endpoint" : "{\"encoded\":\"json\"}"
  },

  "..."
]
```

#### POST /v1/jobs

Request Body:

```json
{
  "name"      : "Job Name",
  "summary"   : "a short description",

  "store"     : "uuid-of-store-to-use",
  "target"    : "uuid-of-target-to-use",
  "retention" : "uuid-of-retention-policy-to-use",
  "schedule"  : "uuid-of-schedule-to-use",

  "paused"    : false
}
```

| Field | Required? | Meaning |
| :---- | :-------: | :------ |
| name | Y | The name of the new job
| summary | N | A short description of the job
| store | Y | The UUID of the store to back data up to
| target | Y | The UUID of the target to back up
| retention | Y | The UUID of the retention policy to apply to backup archives
| schedule | Y | The UUID of the backup schedule to use when determining when this job should run
| paused | Y | Whether or not this job should be paused, initially

Response Body:

```json
{
  "ok"   : "created",
  "uuid" : "6b8398be-fdc0-424a-8532-e812e5dfc116"
}
```

| Field | Meaning |
| :---- | :------ |
| ok    | The new job was created
| uuid  | The UUID of the newly-created job

#### PUT /v1/job/:uuid

Request Body:

```json
{
  "name"      : "Job Name",
  "summary"   : "a short description",

  "store"     : "uuid-of-store-to-use",
  "target"    : "uuid-of-target-to-use",
  "retention" : "uuid-of-retention-policy-to-use",
  "schedule"  : "uuid-of-schedule-to-use"
}
```

| Field | Required? | Meaning |
| :---- | :-------: | :------ |
| name | Y | The name of the new job
| summary | Y | A short description of the job
| store | Y | The UUID of the store to back data up to
| target | Y | The UUID of the target to back up
| retention | Y | The UUID of the retention policy to apply to backup archives
| schedule | Y | The UUID of the backup schedule to use when determining when this job should run

**NOTE:** `summary` is required for update requests, whereas it is optional on creation.

**ALSO NOTE:** The `paused` boolean parameter available on creation is not available
for jobs that already exist.  Use the other `POST` URLs for pausing / unpausing existent
jobs.

Response Body:

```json
{
  "ok" : "updated"
}
```

| Field | Meaning |
| :---- | :------ |
| ok    | The job was updated

#### POST /v1/job/:uuid/run

Request Body:

```json
{
  "owner" : "Username"
}
```

| Field | Required? | Meaning |
| :---- | :-------: | :------ |
| owner | N | Name of the user requesting the job re-run; defaults to "anon"

Response Body:

```json
{
  "ok" : "scheduled"
}
```

| Field | Meaning |
| :---- | :------ |
| ok    | The task was scheduled


### Archive API

Purpose: allows end-users and operators to see what backups have been performed, optionally filtering them to specific targets (just the Cloud Foundry postgres database please), stores (what’s in S3?) and time windows (only show me backups before that data corruption incident).  It also facilitates restoration of data, and purging of backups ahead of schedule.

Note: the PUT /v1/archive/:uuid endpoint is only able to update the annotations (name and summary) for an archive.

| Method | Path | Arguments | Request Body |
| :----- | :---- | :------- | :----------- |
| GET | /v1/archives | ?target=:uuid <br>?store=:uuid <br>?after=YYYYMMDD <br>?before=YYYYMMDD | - |
| GET | /v1/archive/:uuid | - | - |
| POST | /v1/archive/:uuid/restore | { target: $target_uuid } | see below |
| DELETE | /v1/archive/:uuid | - | - |
| PUT | /v1/archive/:uuid | - | see below |

#### GET /v1/archives

```json
[
  {
    "uuid"            : "9ee4b579-19ba-4fa5-94e1-e5b2a4d8e85a",
    "store_key"       : "BKP-1234-56789",

    "taken_at",       : "2015-10-25 11:32:00",
    "expires_at",     : "2015-12-25 11:32:00",
    "notes"           : "a few notes about this archive",

    "store_uuid"      : "b7b5743f-adfa-4ceb-abde-2c2085149b12",
    "store_plugin"    : "store-plugin",
    "store_endpoint"  : "{\"encoded\":\"json\"}",

    "target_uuid"     : "5c7b8b50-ff11-4d67-9624-fd8214bc8629",
    "target_plugin"   : "target-plugin",
    "target_endpoint" : "{\"encoded\":\"json\"}"
  },

  "..."
]
```

#### GET /v1/archive/:uuid

_not yet implemented, apparently_

#### POST /v1/archive/:uuid/restore

Request Body

```json
{
  "target" : "dd322f14-763d-4659-bc49-c2f1f2352341",
  "owner"  : "Username"
}

| Field | Required? | Meaning |
| :---- | :-------: | :------ |
| target | N | UUID of the target to restore this archive to.  Defaults to the target from the original backup job
| owner | N | Username of the user requesting the restoration.  Defaults to "anon"

Response Body:

```json
{
  "ok" : "scheduled"
}
```

| Field | Meaning |
| :---- | :------ |
| ok    | The restore task was scheduled

#### PUT /v1/archive/:uuid

Request Body:

```json
{
  "notes" : "Some notes about this archive"
}
```

| Field | Required? | Meaning |
| :---- | :-------: | :------ |
| notes | Y | Notes about the archive

Response Body:

```json
{
  "ok" : "updated"
}
```

| Field | Meaning |
| :---- | :------ |
| ok    | The archive was updated



### Tasks API
Purpose: allows the Web UI and the CLI to show running tasks, query a specific task, submit new tasks, cancel tasks, etc.

| Method | Path | Arguments | Request Body |
| :----- | :---- | :------- | :----------- |
| GET | /v1/tasks | ?status=:status <br>?debug | - |
| DELETE | /v1/task/:uuid | - | - |

#### GET /v1/tasks

```json
[
  {
    "uuid"         : "5e2c416d-36f7-484a-8a2a-3d3d567d55d6",
    "owner"        : "system",
    "type"         : "backup",

    "job_uuid"     : "274ddd91-6c17-4e5a-b5cd-6d53925d48b4",
    "archive_uuid" : "286102fe-c0fd-4e45-a357-743436a19602",
    "status"       : "done",
    "started_at"   : "2015-11-25 11:30:00",
    "stopped_at"   : "2015-11-25 11:32:00",
    "log"          : "this is the log of the job"
  },

  "..."
]
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
* `race` : runs `ginkgo -race *` to test for race conditions
* `plugin` | `plugins` : builds all the plugin binaries
* `shield` : builds the `shieldd` and `shield-schema` binaries
* `all-the-things` : runs all the tests (except the race test) and builds all the binaries.

`all-the-things` is also the default behavior, so running `make` with no targets is the same as `make all-the-things`.

Examples:

```
$ make shield
go build ./cmd/shieldd
go build ./cmd/shield-schema

$ make tests
ginkgo *
[1447388660] HTTP REST API Test Suite - 69/69 specs ••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••••• SUCCESS! 84.112022ms PASS
[1447388660] Database Layer Test Suite - 19/19 specs ••••••••••••••••••• SUCCESS! 12.447835ms PASS
[1447388660] Plugin Framework Test Suite - 45/45 specs ••••••••••••••••••••••••••••••••••••••••••••• SUCCESS! 22.374368ms PASS
[1447388660] Supervisor Test Suite - 14/14 specs •••••••••••••• SUCCESS! 3.922723257s PASS
[1447388660] Timespec Test Suite - 34/34 specs •••••••••••••••••••••••••••••••••• SUCCESS! 21.766349ms PASS

Ginkgo ran 5 suites in 6.900078754s
Test Suite Passed
```

## CLI Usage Examples

This section is exploratory.

```
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

# "managing" plugins
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
