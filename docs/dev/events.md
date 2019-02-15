SHIELD Events
=============

Central to the SHIELD Web UI interaction is the event stream, a
websocket anchored at /v2/events that relays _events_ from the
backend to the frontend.  This document attempts to catalog and
define each event, where it comes from, and what information is
contained in the event payload.

Queue Naming
------------

Events are dispatched to one or more _queues_, which front-end
users are implicitly subscribed to based on their rights and
privileges inside of SHIELD.

There are a few fixed queus with special uses:

| Queue    | Purpose |
| -------- | ------- |
| `*`      | Everyone gets these messages. |
| `admins` | SHIELD Administrators get these messages. |

Other queues are _parametric_, and use pattern-based names to
represent a multitude of exclusive queues:

| Queue Pattern  | Purpose |
| -------------- | ------- |
| `tenant:$UUID` | Members of the given tenant receive messages. |
| `user:$UUID`   | The given user receives messages. |

Here, `$UUID` represents the lower-case UUID of the object being
targeted, either a tenant or a user.



Event Types
-----------

The following event types are defined, in `$src/core/bus/bus.go`:

| Type                 | Description |
| -------------------- | ----------- |
| `error`              | A general error has occurred.  See **Error Events** |
| `unlock-core`        | Sent when the SHIELD core become unlocked. See **System-wide Events** |
| `create-object`      | Sent when an object (tenant, task, job, etc.) is created. |
| `update-object`      | Sent when an object is updated. |
| `delete-object`      | Sent when an object is deleted. |
| `task-status-update` | Sent when the status of a task is changed. See **Task Events** |
| `task-log-update`    | Sent when new lines in the task log are received by SHIELD. See **Task Events** |
| `tenant-invite`      | Sent when a user is invited into a tenant, or promoted / demoted. See **Tenant Membership Events** |
| `tenant-banish`      | Sent when a user is removed from a tenant. See **Tenant Membership Events** |


Agent Events
------------

When a new Agent is registered with the system, it is sent to the
frontend, as a _create-object_ event, with the following payload:

    {
      "uuid"         : "9145bbe8-c2af-43ca-8cca-95ad1990788d",
      "name"         : "us-east-1/prod-cf/postgres-0",
      "address"      : "10.14.5.6:5444",
      "version"      : "6.7.8",
      "status"       : "failing",
      "last_error"   : "Something broker",
      "last_seen_at" : 1550253924           # UNIX epoch timestamp
    }

This event is sent to the `*` queue, so that all connected web
clients receive it.



Job Events
----------

Events are sent for a job for every step in its lifecycle.  When a
new job is configured, a _create-object_ event for that job is
sent.  Any re-configuration of the job results in an
_update-object_ event, and the removal of a job sends a
_delete-object_ event.

The _create-object_ and _update-object_ events have the following
payload:

    {
      "uuid"        : "44354b93-f509-4838-b7be-007a48c31c5b",
      "name"        : "Daily Backups"
      "summary"     : "A daily backup of the something something",
      "healthy"     : true,
      "keep_n"      : 90,
      "keep_days"   : 90,
      "schedule"    : "daily 3am",
      "paused"      : false,
      "fixed_key"   : false,

      "tenant_uuid" : "bba340d0-2b61-43a9-913f-1be9abdccbd1",
      "target_uuid" : "0e0983fd-92f3-468c-8e9d-92a6d8443f58",
      "store_uuid"  : "2f8bdd7a-fb78-4639-8480-489f78e685ab"
    }

The _delete-object_ event has the following payload:

    {
      "uuid" : "44354b93-f509-4838-b7be-007a48c31c5b"
    }

_(Note: there may be other fields in the _delete-object_ event,
but they are not relevant, and should not be relied upon.)_

All three of these events are sent to the tenant queue
(`tenant:$UUID`).



Store Events
------------

gvents are sent for a store for every step in its lifecycle.  When
a new store is configured, a _create-object_ event for that store
is sent.  Any re-configuration of the store results in an
_update-object_ event, and the removal of a store sends a
_delete-object_ event.

The _create-object_ and _update-object_ events have the following payload:

    {
      "uuid"           : "2f8bdd7a-fb78-4639-8480-489f78e685ab"
      "name"           : "Main S3 Bucket"
      "summary"        : "Our primary bucket in Amazon's S3",
      "agent"          : "127.0.0.1:5444",
      "plugin"         : "s3",
      "healthy"        : true,

      "tenant_uuid"    : "bba340d0-2b61-43a9-913f-1be9abdccbd1",
      "global"         : false,

      "daily_increase" : 43008,              #  42.0 KB
      "storage_used"   : 250295091,          # 238.7 MB
      "threshold"      : 419430400,          # 400.0 MB
      "archive_count"  : 892

      "config" : {
        "bucket" : "backups",
        /* ... etc ... */
      },
      "display_config" : [
        {
          "label" : "S3 Bucket",
          "value" : "bucket"
        },
        /* ... etc ... */
      ]
    }

The _delete-object_ event has the following payload:

    {
      "uuid" : "2f8bdd7a-fb78-4639-8480-489f78e685ab"
    }

_(Note: there may be other fields in the _delete-object_ event,
but they are not relevant, and should not be relied upon.)_

Where these events get sent depends on the visibility of the
store.  Events for global stores are sent to the `*` queue, so
that everyone gets the updates.  Events for tenant-specific stores
are sent to the tenant queue (`tenant:$UUID`).



Target Events
-------------

Events are sent for a target for every step in its lifecycle.
When a new target is configured, a _create-object_ event for that
target is sent.  Any re-configuration of the target results in an
_update-object_ event, and the removal of a target sends a
_delete-object_ event.

The _create-object_ and _update-object_ events have the following payload:

    {
      "uuid"           : "0e0983fd-92f3-468c-8e9d-92a6d8443f58",
      "tenant_uuid"    : "bba340d0-2b61-43a9-913f-1be9abdccbd1",

      "name"           : "Important Files"
      "summary"        : "Our most important of files.",
      "agent"          : "10.0.0.6:5444",
      "plugin"         : "fs",
      "healthy"        : true,
      "compression"    : "bzip2",

      "config" : {
        "root" : "/var/files",
        /* ... etc ... */
      }
    }

The _delete-object_ event has the following payload:

    {
      "uuid" : "0e0983fd-92f3-468c-8e9d-92a6d8443f58",
    }

_(Note: there may be other fields in the _delete-object_ event,
but they are not relevant, and should not be relied upon.)_

These events are sent to the tenant queue (`tenant:$UUID`).



Task Events
-----------

Like other objects, tasks emit events when they are created, and
updated.  Unlike other objects, however, tasks use specific event
types, with custom payloads, for the updates.

When a task is created, a _create-object_ event is sent, with the
following payload:

    {
      "uuid"         : "f6b7689a-e804-4beb-8803-f2b031781f5d",
      "tenant_uuid"  : "bba340d0-2b61-43a9-913f-1be9abdccbd1",
      "owner"        : "system",
      "op"           : "backup",
      "job_uuid"     : "44354b93-f509-4838-b7be-007a48c31c5b",
      "store_uuid"   : "2f8bdd7a-fb78-4639-8480-489f78e685ab"
      "target_uuid"  : "0e0983fd-92f3-468c-8e9d-92a6d8443f58",

      "archive_uuid" : "502e583e-8407-495b-be4a-bd4585993446",
      "status"       : "running",
      "requested_at" : 1550257564,
      "started_at"   : 1550257568,
      "stopped_at"   : 0,
      "ok"           : true,
      "notes"        : "",
      "clear"        : ""
    }

Updates are handled based on the type of update.  Status updates
come through as _task-status-update_ messages with the following
payload:

    {
      "uuid"         : "f6b7689a-e804-4beb-8803-f2b031781f5d",
      "status"       : "done",
      "started_at"   : 1550257568,
      "stopped_at"   : 1550258134,
      "ok"           : true
    }

Whenever text is appended to the task log, usually as a result of
output being received from the agent, a _task-log-update_ message
is sent, with the following payload:

    {
      "uuid" : "f6b7689a-e804-4beb-8803-f2b031781f5d",
      "tail" : "\n\n... an additional log message...\n"
    }

Where these messages are sent depends entirely on the type of
task, and its scope.  Tasks that are specific to a tenant, either
attached to its targets, private cloud storage systems, or
scheduled jobs, are sent to the tenant queue (`tenant:$UUID`).
Tasks that are _not_ tenant-specific are sent to SHIELD
Administrators, via the `admins` queue.



Tenant Events
-------------

There are only two lifecycle events related directly to tenant
objects: `update-object (tenant)` and `delete-object (tenant)`.

_update-object_ events fire whenever anyone makes a change to the
metadata of the tenant itself.  If someone renames a tenant, an
_update-object_ event will fire.  Whenever storage usage
aggregates for the tenant are re-calcualted, the new values will
be communicated through an _update-object_ event.

The contents of a tenant _update-object_ event are:

    {
      "uuid"           : "9c17ffd6-084f-47a7-bf50-75df205a7544",
      "name"           : "Example Tenant",
      "daily_increase" : 43008,              #  42.0 KB
      "storage_used"   : 250295091,          # 238.7 MB
      "archive_count"  : 892
    }

_delete-object_ events fire whenever a SHIELD administrator
removes a tenant (and all of its associated objects) from the
system.

The contents of a tenant _delete-object_ event are:

    {
      "uuid" : "9c17ffd6-084f-47a7-bf50-75df205a7544"
    }

_(Note: there may be other fields in the _delete-object_ event,
but they are not relevant, and should not be relied upon.)_

Both _update-object_ and _delete-object_ events are sent to the
tenant queue (`tenant:$UUID`).



Tenant Membership Events
------------------------

Other important events related to tenants involve membership in
those tenants, specifically, _tenant-invite_ and _tenant-banish_
events.

An _tenant-invite_ event fires when either of the following occurs:

  1. A user is added to a tenant they did not previously have
     access to.

  2. A member of a tenant has their role changed (promoted _or_
     demoted)  on that tenant.

The _tenant-invite_ event payload looks like this:

    {
      "user_uuid"   : "a10a9007-1ece-4b8d-bd9b-dd6b440e8d2a",
      "tenant_uuid" : "9c17ffd6-084f-47a7-bf50-75df205a7544",
      "role"        : "new-role"
    }

A _tenant-banish_ event fires when a member of tenant is removed
from that tenant entirely.  The payload looks like this:

    {
      "user_uuid"   : "a10a9007-1ece-4b8d-bd9b-dd6b440e8d2a",
      "tenant_uuid" : "9c17ffd6-084f-47a7-bf50-75df205a7544"
    }

Both _tenant-invite_ and _tenant-banish_ events are sent to queues
for the tenant (`tenant:$UUID`) and the user (`user:$UUID`).
