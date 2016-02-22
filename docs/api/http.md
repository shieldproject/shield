# HTTP API

## Schedules API

Purpose: allows the Web UI and CLI to find out what schedules are defined, and provides CRUD operations for schedule management.  Allowing queries to filter to unused=t or unused=f enables the frontends to show schedules that can be deleted safely.

| Method | Path | Arguments | Request Body |
| :----- | :---- | :------- | :----------- |
| GET | /v1/schedules | ?unused=[tf] <br> ?name=:search | - |
| POST | /v1/schedules | - | see below |
| DELETE | /v1/schedule/:uuid | - | - |
| GET | /v1/schedule/:uuid | - | - |
| PUT | /v1/schedule/:uuid | - | see below |

### GET /v1/schedules

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

### POST /v1/schedules

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

### PUT /v1/schedule/:uuid

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



## Retention Policies API

Purpose: allows the Web UI and CLI to find out what retention policies are defined, and provides CRUD operations for policy management.  Allowing queries to filter to unused=t or unused=f enables the frontends to show retention policies that can be deleted safely.

| Method | Path | Arguments | Request Body |
| :----- | :---- | :------- | :----------- |
| GET | /v1/retention | ?unused=[tf] <br> ?name=:search | - |
| POST | /v1/retention | - | see below |
| DELETE | /v1/retention/:uuid | - | - |
| GET | /v1/retention/:uuid | - | - |
| PUT | /v1/retention/:uuid | - | see below |

### GET /v1/retention

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

### POST /v1/retention

Request Body:

```json
{
  "name"    : "Policy Name",
  "summary" : "a short description",
  "expires" : 86400
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

### PUT /v1/retention/:uuid

Request Body:

```json
{
  "name"    : "Policy Name",
  "summary" : "a short description",
  "expires" : 86400
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



## Targets API

Purpose: allows the Web UI and CLI to review what targets have been defined, and allows updates to existing targets (to change endpoints or plugins, for example) and remove unused targets (i.e. retired / decommissioned services).

| Method | Path | Arguments | Request Body |
| :----- | :---- | :------- | :----------- |
| GET | /v1/targets | ?plugin=:name <br> ?unused=[tf] <br> ?name=:search  | - |
| POST | /v1/targets | - | see below |
| DELETE | /v1/target/:uuid | - | - |
| GET | /v1/target/:uuid | - | - |
| PUT | /v1/target/:uuid | - | see below |

### GET /v1/targets

```json
[
  {
    "uuid"     : "2f42d0b3-449a-4d0e-8576-a40cc552d7e5",
    "name"     : "Target Name",
    "summary"  : "a short description",
    "plugin"   : "plugin-name",
    "endpoint" : "{\"encoded\":\"json\"}",
    "agent"    : "10.17.66.54:5544"
  },

  "..."
]
```

### POST /v1/targets

Request Body:

```json
{
  "name"     : "Target Name",
  "summary"  : "a short description",
  "plugin"   : "plugin-name",
  "endpoint" : "{\"encoded\":\"json\"}",
  "agent"    : "10.17.66.54:5544"
}
```

| Field | Required? | Meaning |
| :---- | :-------: | :------ |
| name | Y | The name of the new target
| summary | N | A short description of the target
| plugin | Y | The name of the plugin to use when backing up this target
| endpoint | Y | The endpoint configuration required to access this target's data
| agent | Y | The host:port of a Shield agent that can backup/resetore this target

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

### PUT /v1/target/:uuid

Request Body:

```json
{
  "name"     : "Target Name",
  "summary"  : "a short description",
  "plugin"   : "plugin-name",
  "endpoint" : "{\"encoded\":\"json\"}",
  "agent"    : "10.17.66.54:5544"
}
```

| Field | Required? | Meaning |
| :---- | :-------: | :------ |
| name | Y | The name of the new target
| summary | Y | A short description of the target
| plugin | Y | The name of the plugin to use when backing up this target
| endpoint | Y | The endpoint configuration required to access this target's data
| agent | Y | The host:port of a Shield agent that can backup/resetore this target

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



## Stores API

Purpose: allows operators (via the Web UI and CLI components) to view what storage systems are available for configuring backups, provision new ones, update existing ones and delete unused ones.

| Method | Path | Arguments | Request Body |
| :----- | :---- | :------- | :----------- |
| GET | /v1/stores | ?plugin=:name <br> ?unused=[tf] <br> ?name=:search| - |
| POST | /v1/stores | - | see below |
| DELETE | /v1/store/:uuid | - | - |
| GET | /v1/store/:uuid | - | - |
| PUT | /v1/store/:uuid | - | see below |

### GET /v1/stores

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

### POST /v1/stores

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

### PUT /v1/store/:uuid

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



## Jobs API
Purpose: allows end-users and operators to see what jobs have been configured, and the details of those configurations.  The filtering on the main listing / search endpoint (/v1/jobs) allows the frontends to show only jobs for specific schedules (what weekly backups are we running?), retention policies (what backups are we keeping for 90d or more?), and specific targets / stores.

| Method | Path | Arguments | Request Body |
| :----- | :---- | :------- | :----------- |
| GET | /v1/jobs | ?target=:uuid <br> ?store=:uuid <br> ?schedule=:uuid <br> ?retention=:uuid <br> ?paused=[tf] <br> ?name=:search | - |
| POST | /v1/jobs | - | see below |
| DELETE | /v1/job/:uuid | - | - |
| GET | /v1/job/:uuid | - | - |
| PUT | /v1/job/:uuid | - | see below |
| POST | /v1/job/:uuid/pause | - | - |
| POST | /v1/job/:uuid/unpause | - | - |
| POST | /v1/job/:uuid/run | - | see below |

### GET /v1/jobs

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

    "store_uuid"      : "994e991f-112d-496d-a1df-bbdc67c79332",
    "store_plugin"    : "store-plugin",
    "store_endpoint"  : "{\"encoded\":\"json\"}",

    "target_uuid"     : "443e2ce1-de2e-4369-a497-add3dd970d4d",
    "target_plugin"   : "target-plugin",
    "target_endpoint" : "{\"encoded\":\"json\"}"
  },

  "..."
]
```

### POST /v1/jobs

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

### GET /v1/job/:uuid

```json
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

  "store_uuid"      : "994e991f-112d-496d-a1df-bbdc67c79332",
  "store_plugin"    : "store-plugin",
  "store_endpoint"  : "{\"encoded\":\"json\"}",

  "target_uuid"     : "443e2ce1-de2e-4369-a497-add3dd970d4d",
  "target_plugin"   : "target-plugin",
  "target_endpoint" : "{\"encoded\":\"json\"}"
}

```

### PUT /v1/job/:uuid

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

### POST /v1/job/:uuid/run

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


## Archive API

Purpose: allows end-users and operators to see what backups have been performed, optionally filtering them to specific targets (just the Cloud Foundry postgres database please), stores (what’s in S3?) and time windows (only show me backups before that data corruption incident).  It also facilitates restoration of data, and purging of backups ahead of schedule.

Note: the PUT /v1/archive/:uuid endpoint is only able to update the annotations (name and summary) for an archive.

| Method | Path | Arguments | Request Body |
| :----- | :---- | :------- | :----------- |
| GET | /v1/archives | ?target=:uuid  <br> ?store=:uuid  <br> ?after=YYYYMMDD  <br> ?before=YYYYMMDD | - |
| POST | /v1/archive/:uuid/restore | { target: $target_uuid } | see below |
| DELETE | /v1/archive/:uuid | - | - |
| GET | /v1/archive/:uuid | - | - |
| PUT | /v1/archive/:uuid | - | see below |

### GET /v1/archives

```json
[
  {
    "uuid"            : "9ee4b579-19ba-4fa5-94e1-e5b2a4d8e85a",
    "store_key"       : "BKP-1234-56789",

    "taken_at"        : "2015-10-25 11:32:00",
    "expires_at"      : "2015-12-25 11:32:00",
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

### GET /v1/archive/:uuid

_not yet implemented, apparently_

### POST /v1/archive/:uuid/restore

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

### PUT /v1/archive/:uuid

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



## Tasks API

Purpose: allows the Web UI and the CLI to show running tasks, query a specific task, submit new tasks, cancel tasks, etc.

| Method | Path | Arguments | Request Body |
| :----- | :---- | :------- | :----------- |
| GET | /v1/tasks | ?status=:status <br> ?active=[tf] <br> ?debug | - |
| GET | | /v1/task/:uuid | - | - |
| DELETE | /v1/task/:uuid | - | - |

### GET /v1/tasks

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



## Meta API

Purpose: provides public (non-sensitive) information about the
Shield daemon.

| Method | Path | Arguments | Request Body |
| :----- | :---- | :------- | :----------- |
| GET | /v1/meta/pubkey | - | - |

### GET /v1/meta/pubkey

```txt
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC5X75B52xHxfDeujUiKNk9t2jZTR6FIb02t9pUcE6yfwItKGEM8wEad5TVtAqrqdiOaZoosYzcXzzcM2JXsGaCqhVyf2oNaQHiPuyLufPdPW3ZE6omKfHlwL32PkdK4XtZQIwwLEK4NScp1Gvi8GMF90JSaPOQuKgpXCiDXQWFuQkPUzu6yIQIkhPCthtLRn31Td/zF92vBdr5VXyjQ1j8lFTO0jrw9nqwnrW3SA6b1FToSaLvXJJvV8De1Vlkl030tzVdYA4KPIZFX7IPPueVBJcqCaXxEMSzceknGTXP7r64oJDJw4vE39pYqCYtllhzOKKYVaDTHoUUBsZQu+e5 core@shield
```

This can be used by agents to auto-authorize the core daemon for
remote operations, rather than having to specify the key
out-of-band.  There are security risks involved in using this
feature, so be consider the potential for MitM attacks and act
accordingly.


