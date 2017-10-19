# SHIELD v2 API

This document specifies the behavior of the SHIELD API, version 2,
in its entirety.  This is a specification, not merely
documentation &mdash; it is the authoritative source.  If this
document is unclear, it will be amended.  If the SHIELD codebase
disagrees with this specification, the code is incorrect and
should be treated as such.

The purpose of this document is to allow 3rd party integrators to
inter-operate with the SHIELD API without resorting to spelunking
through its implementation.

## Error Handling

The SHIELD API returns errors by using non-200 HTTP status codes,
as follows:

  - **500** - An internal error has occurred; details will be
    present in the SHIELD Core server error log.  The response
    will contain a sanitized error message, using the standard
    format (described below).

  - **400** - Something about the HTTP request was invalid or
    incorrect.  This may occur if a JSON payload was expected,
    but not provided, required keys in theat payload were missing,
    or values supplied were incorrect, out-of-range, etc.

  - **404** - The requested resource was not found.  An error
    (in the standard format) will be returned.

  - **401** - The requester is not authenticated to the SHIELD
    API, but has requested a protected endpoint.  The request
    may be retried after authenticating.

  - **403** - The requester is authenticated but has requested
    an endpoint that they do lack the rights to access.  This
    request should not be retried.

Regardless of the HTTP status code used, the SHIELD API will
always include a JSON payload with more details, in either the
**Standard Format** or the **Missing Values Format**.

### Standard Format for Error Reporting

The **Standard Format** for error reporting consists of a
top-level JSON object containing a single key, `error`, that
contains a human-readable error message, suitable for display.

Example:

```json
{
  "error": "No such retention policy"
}
```

This format is used for all non-validation error reporting.

### Missing Values Format for Error Reporting

The **Missing Values Format** for error reporting is used for
reporting request validation errors where required fields in the
request payload are missing.  It consists of a top-level JSON
object containing a single key, `missing`, which is set to a list
of field names that must be sent in the request, but were not.

Example:

```json
{
  "missing": [
    "name",
    "endpoint",
    "agent"
  ]
}
```

The order of the fields is inconsequential.
## Health

The health endpoints give you a glimpse into the well-being of a
SHIELD Core, for monitoring purposes.


### GET /v2/health

Returns health information about the SHIELD Core, connected
storage accounts, and general metrics.


**Request**

```sh
curl -H 'Accept: application/json' https://shield.host/v2/health
```

**Response**

If all goes well, you will receive a 200 OK, with a `Content-Type`
of `application/json`, and something similar to the following JSON
payload in the response body:

```json
{
  "shield": {
    "version" : "6.7.2",
    "ip"      : "10.0.0.5",
    "fqdn"    : "shield.example.com",
    "env"     : "PRODUCTION",
    "color"   : ""
  },
  "health": {
    "api_ok"     : true,
    "storage_ok" : true,
    "jobs_ok"    : true
  },
  "storage": [
    { "name": "s3", "healthy": true },
    { "name": "fs", "healthy": true } ],
  "jobs": [
    { "target": "BOSH DB", "job": "daily",  "healthy": true },
    { "target": "BOSH DB", "job": "weekly", "healthy": true } ],
  "stats": {
    "jobs"    : 8,
    "systems" : 7,
    "archives": 124,
    "storage" : 243567112,
    "daily"   : 12345000
  }
}
```

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **failed to check SHIELD health**:
  an internal error occurred and should be investigated by the
  site administrators


## SHIELD Authentication

The Authentication endpoints allow clients to authenticate to a
SHIELD Core, providing credentials to prove their identity and
their authorization to perform other tasks inside of SHIELD.


### POST /v2/auth/login

Authenticate against the SHIELD API as a local user, and retrieve
a session ID that can be used for future, authenticated,
interactions.


**Request**

```sh
curl -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -X POST https://shield.host/v2/auth/login \
     --data-binary '
{
  "username": "your-username",
  "password": "your-password"
}'
```

**NOTE:** `password` is sent in cleartext, so SHIELD should aways be
communicating over TLS (HTTPS).

Both fields, `username`, and `password`, are required.

**Response**

```json
{
  "ok": "a-session-id"
}
```

The session ID (return under the `ok` key) should be passed on
subsequent requests as proof of authentication.  This can be done
by setting the `shield7` cookie to the session ID, or by setting
the `X-Shield-Session` request header.

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to log you in**:
  an internal error occurred and should be investigated by the
  site administrators

- **Incorrect username or password**:
  The supplied credentials were incorrect; either the
  user doesn't exist, or the password was wrong.


### GET /v2/auth/logout

Destroy the current session and log the user out.


**Request**

```sh
curl -H 'Accept: application/json' https://shield.host/v2/auth/logout
```

**Response**

```json
{
  "ok" : "Successfully logged out"
}
```

**NOTE:** The same behavior is exhibited when an authenticated
session successfully logs out, as is seen when an unauthenticated
session attempts to log out.

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to log you out**:
  an internal error occurred and should be investigated by the
  site administrators


### GET /v2/auth/id

Retrieve identity and authorization information about the
currently authenticated session.  If the requester has not
authenticated, a suitable response will be returned.


**Request**

```sh
curl -H 'Accept: application/json' https://shield.host/v2/auth/id
```

**Response**

```json
{
  "user": {
    "name"    : "Your Full Name",
    "account" : "username",
    "backend" : "SHIELD",
    "sysrole" : ""
  },
  "tenants": [
    {
      "uuid": "63a8f402-31e6-4503-8fab-66cbcf411ed3",
      "name": "Some Random Tenant",
      "role": "admin"
    },
    {
      "uuid": "860f7685-c311-4ae5-b34d-6991fc721a37",
      "name": "Another Tenant",
      "role": "engineer"
    }
  ],
  "tenant": {
    "uuid": "63a8f402-31e6-4503-8fab-66cbcf411ed3",
    "name": "Some Random Tenant",
    "role": "admin"
  }
}
```

The top-level `user` key contains information about the current
authenticated user, including their name and what authentication
provider they came from.

The `tenants` key lists _all_ of the tenants that this user
belongs to, along with the role assigned on each.  The session is
free to switch between any of these tenants as they see fit.

The `tenant` key contains the tenant definition for the currently
selected tenant, based on user preferences.

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Authentication failed**:
  The request either lacked a session cookie (or an
  `X-Shield-Session` header), or some other internal
  error has occurred, and SHIELD administrators should
  investigate.


## SHIELD Core

These endpoints allow clients to initialize brand new SHIELD
Cores, and unlock or rekey existing ones.


### POST /v2/init

Initializes a new SHIELD Core, to set up the encryption facilities
for storing backup archive encryption keys safely and securely.
Your SHIELD Core can only be initialized once.


**Request**

```sh
curl -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -X POST https://shield.host/v2/init \
     --data-binary '
{
  "master" : "your secret master password"
}'
```

Where:

  - message: master is the plaintext master password
    to use for encrypting the credentials to the
    SHIELD Core storage vault.

**Response**

If all went well, and the SHIELD Core was properly initialized,
you will receive a 200 OK, and the following response:

```json
{
  "ok" : "Successfully initialized the SHIELD Core"
}
```

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to initialize the SHIELD Core**:
  an internal error occurred and should be investigated by the
  site administrators

- **This SHIELD Core has already been initialized**:
  You are attempting to re-initialize a SHIELD Core,
  which is not allowed.


### POST /v2/unlock

Unlocks a SHIELD Core by providing the master password.  This
allows SHIELD to decrypt the keys to access its storage vault and
generate / retrieve backup archvie encryption keys safely and
securely.


**Request**

```sh
curl -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -X POST https://shield.host/v2/unlock \
     --data-binary '
{
  "master" : "your secret master password"
}'
```

- message: master is the plaintext master password
  that was created when you initialized this SHIELD
  Core (or whatever you last rekeyed it to be).

**Response**

On success, you will receive a 200 OK, with the
following response:

```json
{
  "ok" : "Successfully unlocked the SHIELD Core"
}
```

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to unlock the SHIELD Core**:
  an internal error occurred and should be investigated by the
  site administrators

- **This SHIELD Core has not yet been initialized**:
  You may re-attempt this request after initializing
  the SHIELD Core.


### POST /v2/rekey

Changes the master password used for encrypting the credentials
for the SHIELD Core storage vault (where backup archive encryption
keys are held).


**Request**

```sh
curl -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -X POST https://shield.host/v2/rekey \
     --data-binary '
{
  "current" : "your CURRENT master password",
  "new"     : "what you want to change it to"
}'
```

**Response**

If all goes well, you will receive a 200 OK, and the
following response:

```json
{
  "ok" : "Successfully rekeyed the SHIELD core"
}
```

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to rekey the SHIELD Core**:
  an internal error occurred and should be investigated by the
  site administrators


## SHIELD Agents

These endpoints expose information about registered SHIELD Agents.


### GET /v2/agents

Retrieves information about all registered SHIELD Agents.


**Request**

```sh
curl -H 'Accept: application/json' https://shield.host/v2/agents
```

**Response**

```json
{
  "agents": [
    {
      "name"         : "prod/web/42",
      "uuid"         : "1869e296-4aac-4a17-848d-04f73f743326",
      "address"      : "127.0.0.1:5444",
      "version"      : "dev",
      "status"       : "ok",
      "hidden"       : false,
      "last_error"   : "",
      "last_seen_at" : "2017-10-11 18:54:00"
    }
  ],
  "problems": {
    "1869e296-4aac-4a17-848d-04f73f743326": [
      "This SHIELD agent is reporting ..."
    ]
  }
}
```

The top-level `agents` key is a list of object describing each registered agent:

- **name** - The name of the SHIELD Agent, as set by
  the local system administrator (which may not be the
  SHIELD site administrator).

- **uuid** - The internal UUID assigned to this agent by the SHIELD Core.

- **address** - The `host:port` of the agent, from the
  point-of-view of the SHIELD Core.

- **version - The version of the remote SHIELD Agent's software.

- **status - The health status of the remote SHIELD
  Agent, one of `ok` or `failing`.

- **hidden - Whether or not this agent has been administratively hidden.

- **last\_error - TBD

- **last\_seen\_at - When the remote SHIELD Agent last
  made contact with the SHIELD Core to refresh its
  registration and its metadata.  Date is formatted
  YYYY-MM-DD HH:MM:SS, in 24-hour notation.

The top-level `problems` key maps agent UUIDs to a list of errors detected
statically by the SHIELD Core software.  Each problem is represented as an
English-language description of the underlying issue.  SHIELD reports these
problems to assist site administrators who may be running heterogenous
versions of the SHIELD Core and SHIELD Agent software.  In these
environments, issues may arise due to version incompatibility.  Newer
versions of the SHIELD Core may also be able to inform administrators about
known deficiencies in older version of the SHIELD Agent and SHIELD plugins.

**NOTE:** `problems` are reported by the SHIELD Core; it is perfectly
acceptable for an agent to report itself as healthy, but for the SHIELD Core
to assert that a problem exists.

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to retrieve agent information**:
  an internal error occurred and should be investigated by the
  site administrators


### POST /v2/agents

Initiate agent registration.  The client must supply a POST body
with the details of the agent being registered.

Once an agent has pre-registered, the SHIELD Core will schedule a
"pingback" task, connecting to the agent using its remote peer
address (from the registration HTTP conversation) and the given
port.  This pingback occurs via the SSH protocol, with an op type
of "ping".  The agent must respond with the same _name_ that it
sent in the registration.

This exchange allows the SHIELD to validate registration requests,
using a weak form of authentication.


**Request**

```sh
curl -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -X POST https://shield.host/v2/agents \
     --data-binary '
{
  "name" : "some-identifier",
  "port" : 5444
}'
```

Where:

- message: name is the name of the agent to display in
  the backend, and in log messages.  Usually, an FQDN
  or other unique host identifier is preferable here.
- message: port is the port number that the SHIELD
  agent is bound to.  The remote peer IP will be
  determined from the HTTP request's peer address.

**Response**

On success, you will receive a 200 OK, and the following response:

```json
{
  "ok" : "Pre-registered agent <name> at <host>:<port>"
}
```

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **No \`name' provided with pre-registration request**:
  Your request was missing the required `name`
  argument.  Re-attempt with the `name` argument.

- **No \`port' provided with pre-registration request**:
  Your request was missing the required `port`
  argument.  Re-attempt with the `port` argument.

- **Unable to pre-register agent \<name\> at \<host\>:\<port\>**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unable to determine remote peer address from '\<peer\>'**:
  SHIELD was unable to parse the HTTP connection's
  peer address as a valid IP address.  This should be
  investigated by the site administrators, your local
  network administrator, and possibly the SHIELD
  development team.


### GET /v2/agents/:uuid

Retrieve extended information about a single SHIELD Agent, including its
plugin metadata (what plugins are present, what configuration they accept or
require, etc.)


**Request**

```sh
curl -H 'Accept: application/json' https://shield.host/v2/agents/:uuid
```

**Response**

```json
{
  "agent": {
    "name"         : "prod/web/42",
    "uuid"         : "1869e296-4aac-4a17-848d-04f73f743326",
    "address"      : "127.0.0.1:5444",
    "version"      : "dev",
    "status"       : "ok",
    "hidden"       : false,
    "last_error"   : "",
    "last_seen_at" : "2017-10-11 18:54:00"
  },
  "metadata": {
    "name"    : "prod/web/42",
    "version" : "dev",
    "health"  : "ok",

    "plugins": {
      "fs": {
        "author"   : "Stark & Wayne",
        "features" : {
          "store"  : "yes",
          "target" : "no"
        },

        "fields": [
          {
            "mode"     : "store",
            "name"     : "storage_account",
            "title"    : "Storage Account",
            "help"     : "Name of the Azure Storage Account for accessing the blobstore.",
            "type"     : "string",
            "required" : true
          }
        ]
      }
    }
  },
  "problems": [
    "This SHIELD agent is reporting ..."
  ]
}
```

The top-level `agents` key contains the same agent
information that the `GET /v2/agents` endpoint
returns.  Similarly, the `problems` key contains the
list of issues the SHIELD Core detected, based on this
agent's configuration / version.

The `metadata` key is exclusive to this endpoint, and
contains all of the agent metadata.  Of particular
interest is the `plugins` key, which contains a map of
plugin metadata, keyed by the plugin name.  The format
of this metadata is documented in
[/docs/plugins.md](/docs/plugins.md).

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to retrieve agent information**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such agent**:
  The requested agent UUID was not found in the list
  of registered agents.


## SHIELD Tenants

Tenants serve to insulate groups of SHIELD users from one another,
providing them a virtual view of SHIELD resources.  Each tenant
has their own targets, stores, and retention policy definitions,
as well as their own job configurations.  Each tenants archives
and tasks are visible only to members of that tenant, pursuant to
their assigned roles.


### GET /v2/tenants

Retrieve the list of all tenants currently defined.


**Request**

```sh
curl -H 'Accept: application/json' https://shield.host/v2/tenants
```

**Response**

```json
[
  {
    "name": "A Tenant",
    "uuid": "f2ebbb9f-87f9-43e0-8515-dfce5d4d844c"
  },
  {
    "name": "Some Other Tenant",
    "uuid": "4b6f6e2a-6ac6-443e-a910-aa412744165e"
  }
]
```

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to retrieve tenants information**:
  an internal error occurred and should be investigated by the
  site administrators


### POST /v2/tenants

Create a new tenant.


**Request**

```sh
curl -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -X POST https://shield.host/v2/tenants \
     --data-binary '
{
  "name"  : "New Tenant Name",
  "users" : [
    {
      "uuid"    : "989b724b-bd3d-4799-bfbd-75b2fb5b41f3",
      "account" : "juser",
      "role"    : "engineer"
    },
    {
      "uuid"    : "96d24e33-8e57-4431-95fb-f18b9dfa319a",
      "account" : "jhunt",
      "role"    : "operator"
    }
  ]
}'
```

The `name` field is required.

The `users` list contains a list of initial tenant
role assignments.  The `account` key of each user
object is optional, but can assist site administrators
when troubleshooting assignment issues (since it will
be printed to the log) -- integrations are encouraged
to always send it.

The `role` field indicates what level of access to
grant each invitee, and must be one of:

  - **admin** - Full administrative control, including the ability to add
    and remove users from the tenant, and change role assignments.  Use with
    caution.

  - **engineer** - Full configuration control, including the ability to
    create, update, and delete targets, stores, and retention policies.

  - **operator** - Operational access for running ad hoc backup jobs,
    pausing and unpausing defined jobs, and performing restores.

**Response**

```json
{
  "name": "A New Tenant",
  "uuid": "52d20ef4-f154-431e-a5bb-bb3a200976bb"
}
```

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to creeate new tenant**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unrecognized user account**:
  The request indicated a tenant invitation to a user
  account that was not found in the SHIELD database.
  The request should not be retried.

- **Unable to invite $user to tenant $tenant - only
local users can be invited.
**:
  The request indicated a tenant invitation to a user
  account that was created by a non-local
  authentication provider (i.e. Github).  Tenant
  assignments for 3rd party accounts are governed
  solely by their corresponding authentication
  provider configuration.  The request should not be
  retried.

- **Unable to invite $user to tenant $tenant**:
  an internal error occurred and should be investigated by the
  site administrators


### GET /v2/tenants/:uuid

Request more detailed information about a single tenant.


**Request**

```sh
curl -H 'Accept: application/json' https://shield.host/v2/tenants/:uuid
```

**Response**

```json
{
  "name": "A Tenant",
  "uuid": "f2ebbb9f-87f9-43e0-8515-dfce5d4d844c",

  "members": [
    {
      "uuid"    : "5cb299bf-217f-4756-8eaa-e8a47865869e",
      "account" : "jhunt",
      "name"    : "James Hunt",
      "backend" : "local",
      "role"    : "admin",
      "sysrole" : ""
    }
  ]
}
```

The `members` key will be absent if this tenant has no members.

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to retrieve tenant information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unable to retrieve tenant memberships information**:
  an internal error occurred and should be investigated by the
  site administrators


### PUT /v2/tenants/:uuid

Update a tenant with new attributes.


**Request**

```sh
curl -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -X PUT https://shield.host/v2/tenants/:uuid \
     --data-binary '
{
  "name" : "A New Name"
}'
```

**Response**

```json
{
  "name" : "A New Name",
  "uuid" : "adcfee48-8b43-4ba3-9438-e0da55b8e9df"
}
```

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to update tenant**:
  an internal error occurred and should be investigated by the
  site administrators


### POST /v2/tenants/:uuid/invite

Invite one or more local users to a tenant.


**Request**

```sh
curl -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -X POST https://shield.host/v2/tenants/:uuid/invite \
     --data-binary '
{
  "users": [
    {
      "uuid"    : "5cb299bf-217f-4756-8eaa-e8a47865869e",
      "account" : "jhunt",
      "role"    : "operator"
    },
    {
      "uuid"    : "c608cc65-b134-4581-9bdc-1fa3d0367961",
      "account" : "tmitchell",
      "role"    : "engineer"
    }
  ]
}'
```

Even if you only need to invite a single user, you must specify a list of
user objects.

**Response**

```json
{
  "ok" : "Invitations sent"
}
```

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to retrieve tenant information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unrecognized user account**:
  The request indicated a tenant invitation to a user
  account that was not found in the SHIELD database.
  The request should not be retried.

- **Unable to invite $user to tenant $tenant - only
local users can be invited
**:
  The request indicated a tenant invitation to a user
  account that was created by a non-local
  authentication provider (i.e. Github).  Tenant
  assignments for 3rd party accounts are governed
  solely by their corresponding authentication
  provider configuration.  The request should not be
  retried.

- **Unable to invite $user to tenant $tenant**:
  an internal error occurred and should be investigated by the
  site administrators


### POST /v2/tenants/:uuid/banish

Remove a user from a tenant they currently belong to.


**Request**

```sh
curl -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -X POST https://shield.host/v2/tenants/:uuid/banish \
     --data-binary '
{
  "users": [
    {
      "uuid"    : "20d5bd91-9f7b-4551-9279-8571b8292003",
      "account" : "gfranks"
    },
    {
      "uuid"    : "c608cc65-b134-4581-9bdc-1fa3d0367961",
      "account" : "tmitchell"
    }
  ]
}'
```

**Response**

```json
{
  "ok" : "Banishments served."
}
```

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to retrieve tenant information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unrecognized user account**:
  The request indicated a tenant banishment of a user
  account that was not found in the SHIELD database.
  The request should not be retried.

- **Unable to banish $user to tenant $tenant - only
local users can be banished
**:
  The request indicated a tenant banishment of a user
  account that was created by a non-local
  authentication provider (i.e. Github).  Tenant
  assignments for 3rd party accounts are governed
  solely by their corresponding authentication
  provider configuration.  The request should not be
  retried.

- **Unable to banish $user to tenant $tenant**:
  an internal error occurred and should be investigated by the
  site administrators


## SHIELD Targets

Targets represent the data systems that SHIELD runs backup and
restore operations against as course of normal function.


### GET /v2/tenants/:tenant/targets

Retrieve all defined targets for a tenant.


**Request**

```sh
curl -H 'Accept: application/json' https://shield.host/v2/tenants/:tenant/targets
```

**Response**

```json
[
  {
    "uuid"     : "b4400ee0-dce9-4277-9948-02a56ad51b17",
    "name"     : "Some Target",
    "summary"  : "The operator-supplied description of this target",
    "agent"    : "127.0.0.1:5444",
    "endpoint" : "{}",
    "plugin"   : "fs"
  }
]
```

**NOTE:** the `endpoint` key is currently a string of JSON, which
means that it contains lots of escape sequences.  Future versions
of the v2 API (prior to launch) may alter this to be the full
JSON, inline, for both readability and sanity's sake.

FIXME: Fix target.endpoint string -> JSON problem.

**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `operator` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve targets information**:
  an internal error occurred and should be investigated by the
  site administrators


### POST /v2/tenants/:tenant/targets

Create a new target in a tenant.


**Request**

```sh
curl -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -X POST https://shield.host/v2/tenants/:tenant/targets \
     --data-binary '
{
  "name"     : "New Target Name",
  "summary"  : "A longer description of the target",
  "agent"    : "127.0.0.1:5444",
  "endpoint" : "{}",
  "plugin"   : "plugin"
}'
```

Note: the `endpoint` key is currently a string of JSON, which
means that it contains lots of escape sequences.  Future versions
of the v2 API (prior to launch) may alter this to be the full
JSON, inline, for both readability and sanity's sake.

FIXME: Fix target.endpoint string -> JSON problem.

**Response**

```json
{
  "uuid"     : "b6d03df5-6978-43d8-ad9e-a22f8ec8457a",
  "name"     : "New Target Name",
  "summary"  : "A longer description of the target",
  "agent"    : "127.0.0.1:5444",
  "endpoint" : "{}",
  "plugin"   : "plugin"
}
```

Note: the `endpoint` key is currently a string of JSON, which
means that it contains lots of escape sequences.  Future versions
of the v2 API (prior to launch) may alter this to be the full
JSON, inline, for both readability and sanity's sake.

FIXME: Fix target.endpoint string -> JSON problem.

**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `engineer` role on the tenant.

**Errors**

The following error messages can be returned:

- **Authorization required**:
  The request was made without an authenticated
  session or auth token.  See **Authentication** for
  more details.  The request may be retried after
  authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role
  assignment.  The request should not be retried.

- **Unable to retrieve tenant information**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such tenant**:
  No tenant was found with the given UUID.
- **Unable to create new data target**:
  an internal error occurred and should be investigated by the
  site administrators


### GET /v2/tenants/:tenant/targets/:uuid

Retrieve a single target for a tenant.


**Request**

```sh
curl -H 'Accept: application/json' https://shield.host/v2/tenants/:tenant/targets/:uuid
```

**Response**

```json
{
  "uuid"     : "b4400ee0-dce9-4277-9948-02a56ad51b17",
  "name"     : "Some Target",
  "summary"  : "The operator-supplied description of this target",
  "agent"    : "127.0.0.1:5444",
  "endpoint" : "{}",
  "plugin"   : "fs"
}
```

Note: the `endpoint` key is currently a string of JSON, which
means that it contains lots of escape sequences.  Future versions
of the v2 API (prior to launch) may alter this to be the full
JSON, inline, for both readability and sanity's sake.

FIXME: Fix target.endpoint string -> JSON problem.

**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `operator` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve tenant information**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such target**:
  No target with the given UUID exists on the
  specified tenant.


### PUT /v2/tenants/:tenant/targets/:uuid

Update an existing target on a tenant.


**Request**

```sh
curl -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -X PUT https://shield.host/v2/tenants/:tenant/targets/:uuid \
     --data-binary '
{
  "name"     : "Updated Target Name",
  "summary"  : "A longer description of the target",
  "agent"    : "127.0.0.1:5444",
  "endpoint" : "{}",
  "plugin"   : "plugin"
}'
```

You can specify as many or few of these fields as you want;
omitted fields will be left at their previous values.

Note: the `endpoint` key is currently a string of JSON, which
means that it contains lots of escape sequences.  Future versions
of the v2 API (prior to launch) may alter this to be the full
JSON, inline, for both readability and sanity's sake.

FIXME: Fix target.endpoint string -> JSON problem.

**Response**

```json
{
  "ok" : "Updated target successfully"
}
```

**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `engineer` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve tenant information**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such target**:
  No target with the given UUID exists on the
  specified tenant.

- **Unable to update target**:
  No target with the given UUID exists on the
  specified tenant.


### DELETE /v2/tenants/:tenant/targets/:uuid

Remove a target from a tenant.


**Request**

```sh
curl -H 'Accept: application/json' \
     -X DELETE https://shield.host/v2/tenants/:tenant/targets/:uuid \
```

**Response**

```json
{
  "ok": "Target deleted successfully"
}
```

**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `engineer` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve tenant information**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such target**:
  No target with the given UUID exists on the
  specified tenant.

- **Unable to delete target**:
  an internal error occurred and should be investigated by the
  site administrators

- **The target cannot be deleted at this time**:
  This target is referenced by one or more extant job
  configuration; deleting it would lead to an
  incomplete (and unusable) setup.


## SHIELD Stores

Storage systems are essential to any data protection efforts,
since the protected data must reside elsewhere, on another system
in order to be truly safe.  Stores provide definitions of external
storage system where backup archives will be kept.

**NOTE:** the API endpoints in this section deal exclusively with
tenant-scoped storage systems.  For information on the endpoints
for managing global storage solutions, see the section titled
**SHIELD Global Resources**.


### GET /v2/tenants/:tenant/stores

Retrieve all defined stores for a tenant.


**Request**

```sh
curl -H 'Accept: application/json' https://shield.host/v2/tenants/:tenant/stores
```

**Response**

```json
[
  {
    "uuid"    : "925c83ad-22e6-4cdd-bf63-6dd6d09cd86f",
    "name"    : "Cloud Storage Name",
    "summary" : "A longer description of the storage configuration",
    "agent"   : "127.0.0.1:5444",
    "plugin"  : "fs",
    "config"  : {
      "base_dir" : "/var/data/root",
      "bsdtar"   : "bsdtar"
    }
  }
]
```

The values under `config` will depend entirely on what the
operator specified when they initially configured the storage
system.

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to retrieve storage systems information**:
  an internal error occurred and should be investigated by the
  site administrators


### POST /v2/tenants/:tenant/stores

Create a new store on a tenant.


**Request**

```sh
curl -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -X POST https://shield.host/v2/tenants/:tenant/stores \
     --data-binary '
{
  "name"    : "Storage System Name",
  "summary" : "A longer description for this storage system.",
  "plugin"  : "plugin-name",
  "agent"   : "127.0.0.1:5444",
  "config"  : {
    "plugin-specific": "configuration"
  }
}'
```

The values under `config` will depend entirely on which `plugin`
has been selected; no validation will be done by the SHIELD Core,
until the storage system is used in a job.

**Response**

```json
{
  "name"    : "Storage System Name",
  "summary" : "A longer description for this storage system.",
  "plugin"  : "plugin-name",
  "agent"   : "127.0.0.1:5444",
  "config"  : {
    "plugin-specific": "configuration"
  }
}
```

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to retrieve storage system information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unable to create new storage system**:
  an internal error occurred and should be investigated by the
  site administrators


### GET /v2/tenants/:tenant/stores/:uuid

Retrieve a single store for the given tenant.


**Request**

```sh
curl -H 'Accept: application/json' https://shield.host/v2/tenants/:tenant/stores/:uuid
```

**Response**

```json
{
  "uuid"    : "925c83ad-22e6-4cdd-bf63-6dd6d09cd86f",
  "name"    : "Cloud Storage Name",
  "summary" : "A longer description of the storage configuration",
  "plugin"  : "fs",
  "agent"   : "127.0.0.1:5444",
  "config"  : {
    "base_dir" : "/var/data/root",
    "bsdtar"   : "bsdtar"
  }
}
```

The values under `config` will depend entirely on what the
operator specified when they initially configured the storage
system.

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to retrieve storage system information**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such storage system**:
  No storage system with the given UUID exists on the
  specified tenant.


### PUT /v2/tenants/:tenant/stores/:uuid

Update an existing store on a tenant.


**Request**

```sh
curl -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -X PUT https://shield.host/v2/tenants/:tenant/stores/:uuid \
     --data-binary '
{
  "name"    : "Updated Store Name",
  "summary" : "A longer description of the storage system",
  "agent"   : "127.0.0.1:5444",
  "plugin"  : "plugin",
  "config"  : {
    "new": "plugin configuration"
  }
}'
```

You can specify as many or few of these fields as you want;
omitted fields will be left at their previous values.  If `config`
is supplied, it will overwrite the value currently in the
database.

The values under `config` will depend entirely on which `plugin`
has been selected; no validation will be done by the SHIELD Core,
until the storage system is used in a job.

**Response**

```json
{
  "name"    : "Updated Store Name",
  "summary" : "A longer description of the storage system",
  "agent"   : "127.0.0.1:5444",
  "plugin"  : "plugin",
  "config"  : {
    "new": "plugin configuration"
  }
}
```

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to retrieve storage system information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unable to update storage system**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such storage system**:
  No storage system with the given UUID exists on the
  specified tenant.


### DELETE /v2/tenants/:tenant/stores/:uuid

Remove a store from a tenant.


**Request**

```sh
curl -H 'Accept: application/json' \
     -X DELETE https://shield.host/v2/tenants/:tenant/stores/:uuid \
```

**Response**

```json
{
  "ok": "Storage system deleted successfully"
}
```

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to retrieve storage system information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unable to update storage system**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unable to delete storage system**:
  an internal error occurred and should be investigated by the
  site administrators

- **The storage system cannot be deleted at this time**:
  This storage system is referenced by one or more
  extant job configuration; deleting it would lead to
  an incomplete (and unusable) setup.


## SHIELD Retention Policies

Retention Policies govern how long backup archives are kept,
to ensure that storage usage doesn't continue to increase
inexorably.


### GET /v2/tenants/:tenant/policies

Retrieve all defined retention policies for a tenant.


**Request**

```sh
curl -H 'Accept: application/json' https://shield.host/v2/tenants/:tenant/policies
```

**Response**

```json
[
  {
    "uuid"    : "f4dedf80-cdb2-4c81-9a58-b3a8282e3202",
    "name"    : "Long-Term Storage",
    "summary" : "A long-term solution, for infrequent backups only.",
    "expires" : 7776000
  }
]
```

The `expires` key is specified in seconds, but must
always be a multiple of 86400 (1 day).

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to retrieve retention policies information**:
  an internal error occurred and should be investigated by the
  site administrators


### POST /v2/tenants/:tenant/policies

Create a new retention policy in a tenant.


**Request**

```sh
curl -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -X POST https://shield.host/v2/tenants/:tenant/policies \
     --data-binary '
{
  "name"    : "Retention Policy Name",
  "summary" : "A longer description of the policy",
  "expires" : 86400
}'
```

The `expires` value must be specified in seconds, and
must be at least 86,400 (1 day) and be a multiple of
86,400.

**Response**

```json
{
  "uuid"    : "4882b332-6182-4123-984f-f9e5dd8dae20",
  "name"    : "Retention Policy Name",
  "summary" : "A longer description of the policy",
  "expires" : 86400
}
```

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Retention policy expiry must be greater than 1 day**:
  You supplied an `expires` value less than 86,400.
  Please re-try the request with a higher value.

- **Retention policy expire must be a multiple of 1 day**:
  You supplied an `expires` value that was not a
  multiple of 86,400.  Please re-try the request with
  a different value.

- **Unable to create retention policy**:
  an internal error occurred and should be investigated by the
  site administrators


### GET /v2/tenants/:tenant/policies/:uuid

Retrieve a single retention policy for a tenant.


**Request**

```sh
curl -H 'Accept: application/json' https://shield.host/v2/tenants/:tenant/policies/:uuid
```

**Response**

```json
{
  "uuid"    : "4882b332-6182-4123-984f-f9e5dd8dae20",
  "name"    : "Retention Policy Name",
  "summary" : "A longer description of the policy",
  "expires" : 86400
}
```

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to retrieve retention policy information**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such retention policy**:
  No retention policy with the given UUID exists on
  the specified tenant.


### PUT /v2/tenants/:tenant/policies/:uuid

Update a single retention policy on a tenant.


**Request**

```sh
curl -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -X PUT https://shield.host/v2/tenants/:tenant/policies/:uuid \
     --data-binary '
{
  "name"    : "Updated Retention Policy Name",
  "summary" : "A longer description of the retention policy",
  "expires" : 86400
}'
```

You can specify as many or few of these fields as you want;
omitted fields will be left at their previous values.

**Response**

```json
{
  "uuid"    : "4882b332-6182-4123-984f-f9e5dd8dae20",
  "name"    : "Retention Policy Name",
  "summary" : "A longer description of the policy",
  "expires" : 86400
}
```

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Retention policy expiry must be greater than 1 day**:
  You supplied an `expires` value less than 86,400.
  Please re-try the request with a higher value.

- **Retention policy expire must be a multiple of 1 day**:
  You supplied an `expires` value that was not a
  multiple of 86,400.  Please re-try the request with
  a different value.

- **Unable to update retention policy**:
  an internal error occurred and should be investigated by the
  site administrators


### DELETE /v2/tenants/:tenant/policies/:uuid

Remove a retention policy from a tenant.


**Request**

```sh
curl -H 'Accept: application/json' \
     -X DELETE https://shield.host/v2/tenants/:tenant/policies/:uuid \
```

**Response**

```json
{
  "ok": "Retention policy deleted successfully"
}
```

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to retrieve retention policy information**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such retention policy**:
  No retention policy with the given UUID exists on
  the specified tenant.

- **Unable to delete retention policy**:
  an internal error occurred and should be investigated by the
  site administrators

- **The retention policy cannot be deleted at this time**:
  This retention policy is referenced by one or more
  extant job configuration; deleting it would lead to
  an incomplete (and unusable) setup.


## SHIELD Jobs

Jobs are discrete, schedulable backup operations that tie together a
target system, a cloud storage system, a schedule, and a retention
policies.  Without Jobs, SHIELD cannot actually back anything up.


## SHIELD Tasks

Tasks represent the context, status, and output of the execution of
some operation, be it a scheduled backup job being run, an ad hoc
restore operation, etc.

Since tasks represent internal state, they cannot easily be created or
updated via operators.  Instead, the execution of the job, or the
triggering of some other action, will cause a task to spring into
existence and move through its lifecycle.


### DELETE /v2/tenants/:tenant/tasks/:uuid

Cancel a task.


**Request**

```sh
curl -H 'Accept: application/json' \
     -X DELETE https://shield.host/v2/tenants/:tenant/tasks/:uuid \
```

**Response**

```json
{
  "ok": "Task canceled successfully"
}
```

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

This API endpoint does not return any error conditions.

## SHIELD Backup Archives

Each backup archive contains the complete set of data retrieved from a
target system during a single backup job task.

Archives cannot be created _a priori_ via the API alone.  New archives
will be created from scheduled and ad hoc backup jobs, when they
succeed.


## SHIELD Global Resources

Some resources are shared between tenants, either implicitly via
copying (like retention policies), or explicitly (like shared
storage system definitions).


### GET /v2/global/stores

Retrieve all globally-defined stores.


**Request**

```sh
curl -H 'Accept: application/json' https://shield.host/v2/global/stores
```

**Response**

```json
[
  {
    "uuid"    : "925c83ad-22e6-4cdd-bf63-6dd6d09cd86f",
    "name"    : "Cloud Storage Name",
    "summary" : "A longer description of the storage configuration",
    "agent"   : "127.0.0.1:5444",
    "plugin"  : "fs",
    "config"  : {
      "base_dir" : "/var/data/root",
      "bsdtar"   : "bsdtar"
    }
  }
]
```

The values under `config` will depend entirely on what the
operator specified when they initially configured the storage
system.

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to retrieve storage systems information**:
  an internal error occurred and should be investigated by the
  site administrators


### POST /v2/global/stores

Create a new shared storage system.  This storage will be visible
to all tenants.


**Request**

```sh
curl -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -X POST https://shield.host/v2/global/stores \
     --data-binary '
{
  "name"    : "Storage System Name",
  "summary" : "A longer description for this storage system.",
  "plugin"  : "plugin-name",
  "agent"   : "127.0.0.1:5444",
  "config"  : {
    "plugin-specific": "configuration"
  }
}'
```

The values under `config` will depend entirely on which `plugin`
has been selected; no validation will be done by the SHIELD Core,
until the storage system is used in a job.

**Response**

```json
{
  "name"    : "Storage System Name",
  "summary" : "A longer description for this storage system.",
  "plugin"  : "plugin-name",
  "agent"   : "127.0.0.1:5444",
  "config"  : {
    "plugin-specific": "configuration"
  }
}
```

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to retrieve storage system information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unable to create new storage system**:
  an internal error occurred and should be investigated by the
  site administrators


### GET /v2/global/stores/:uuid

Retrieve a single globally-defined storage system.


**Request**

```sh
curl -H 'Accept: application/json' https://shield.host/v2/global/stores/:uuid
```

**Response**

```json
{
  "uuid"    : "925c83ad-22e6-4cdd-bf63-6dd6d09cd86f",
  "name"    : "Cloud Storage Name",
  "summary" : "A longer description of the storage configuration",
  "plugin"  : "fs",
  "agent"   : "127.0.0.1:5444",
  "config"  : {
    "base_dir" : "/var/data/root",
    "bsdtar"   : "bsdtar"
  }
}
```

The values under `config` will depend entirely on what the
operator specified when they initially configured the storage
system.

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to retrieve storage system information**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such storage system**:
  No storage system with the given UUID exists
  (globally).


### PUT /v2/global/stores/:uuid

Update an existing globally-defined storage system.


**Request**

```sh
curl -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -X PUT https://shield.host/v2/global/stores/:uuid \
     --data-binary '
{
  "name"    : "Updated Store Name",
  "summary" : "A longer description of the storage system",
  "agent"   : "127.0.0.1:5444",
  "plugin"  : "plugin",
  "config"  : {
    "new": "plugin configuration"
  }
}'
```

You can specify as many or few of these fields as you want;
omitted fields will be left at their previous values.  If `config`
is supplied, it will overwrite the value currently in the
database.

The values under `config` will depend entirely on which `plugin`
has been selected; no validation will be done by the SHIELD Core,
until the storage system is used in a job.

**Response**

```json
{
  "name"    : "Updated Store Name",
  "summary" : "A longer description of the storage system",
  "agent"   : "127.0.0.1:5444",
  "plugin"  : "plugin",
  "config"  : {
    "new": "plugin configuration"
  }
}
```

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to retrieve storage system information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unable to update storage system**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such storage system**:
  No storage system with the given UUID exists
  (globally).


### DELETE /v2/global/stores/:uuid

Remove a globally-defined storage system.


**Request**

```sh
curl -H 'Accept: application/json' \
     -X DELETE https://shield.host/v2/global/stores/:uuid \
```

**Response**

```json
{
  "ok": "Storage system deleted successfully"
}
```

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to retrieve storage system information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unable to update storage system**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unable to delete storage system**:
  an internal error occurred and should be investigated by the
  site administrators

- **The storage system cannot be deleted at this time**:
  This storage system is referenced by one or more
  extant job configuration; deleting it would lead to
  an incomplete (and unusable) setup.


### GET /v2/global/policies

Retrieve all defined retention policy templates.


**Request**

```sh
curl -H 'Accept: application/json' https://shield.host/v2/global/policies
```

**Response**

```json
[
  {
    "uuid"    : "f4dedf80-cdb2-4c81-9a58-b3a8282e3202",
    "name"    : "Long-Term Storage",
    "summary" : "A long-term solution, for infrequent backups only.",
    "expires" : 7776000
  }
]
```

The `expires` key is specified in seconds, but must always be a
multiple of 86400 (1 day).

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to retrieve retention policy templates information**:
  an internal error occurred and should be investigated by the
  site administrators


### POST /v2/global/policies

Create a new retention policy template.


**Request**

```sh
curl -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -X POST https://shield.host/v2/global/policies \
     --data-binary '
{
  "name"    : "Retention Policy Name",
  "summary" : "A longer description of the policy",
  "expires" : 86400
}'
```

The `expires` value must be specified in seconds, and must be at
least 86,400 (1 day) and be a multiple of 86,400.

**Response**

```json
{
  "uuid"    : "4882b332-6182-4123-984f-f9e5dd8dae20",
  "name"    : "Retention Policy Name",
  "summary" : "A longer description of the policy",
  "expires" : 86400
}
```

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Retention policy expiry must be greater than 1 day**:
  You supplied an `expires` value less than 86,400.
  Please re-try the request with a higher value.

- **Retention policy expire must be a multiple of 1 day**:
  You supplied an `expires` value that was not a
  multiple of 86,400.  Please re-try the request with
  a different value.

- **Unable to create retention policy template**:
  an internal error occurred and should be investigated by the
  site administrators


### GET /v2/global/policies/:uuid

Retrieve a single retention policy template.


**Request**

```sh
curl -H 'Accept: application/json' https://shield.host/v2/global/policies/:uuid
```

**Response**

```json
{
  "uuid"    : "4882b332-6182-4123-984f-f9e5dd8dae20",
  "name"    : "Retention Policy Name",
  "summary" : "A longer description of the policy",
  "expires" : 86400
}
```

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to retrieve retention policy template information**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such retention policy template**:
  No retention policy template with the given UUID
  exists globally.


### PUT /v2/global/policies/:uuid

Update a single retention policy template.


**Request**

```sh
curl -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -X PUT https://shield.host/v2/global/policies/:uuid \
     --data-binary '
{
  "name"    : "Updated Retention Policy Name",
  "summary" : "A longer description of the retention policy",
  "expires" : 86400
}'
```

You can specify as many or few of these fields as you want;
omitted fields will be left at their previous values.

**NOTE:** Updating a retention policy template will not affect any
tenants created prior to the update; updates will apply to new,
future tenants.

**Response**

```json
{
  "uuid"    : "4882b332-6182-4123-984f-f9e5dd8dae20",
  "name"    : "Retention Policy Name",
  "summary" : "A longer description of the policy",
  "expires" : 86400
}
```

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to retrieve retention policy template information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Retention policy expiry must be greater than 1 day**:
  You supplied an `expires` value less than 86,400.
  Please re-try the request with a higher value.

- **Retention policy expire must be a multiple of 1 day**:
  You supplied an `expires` value that was not a
  multiple of 86,400.  Please re-try the request with
  a different value.

- **Unable to update retention policy template**:
  an internal error occurred and should be investigated by the
  site administrators


### DELETE /v2/global/policies/:uuid

Remove a retention policy template.

**NOTE:** Removing a retention policy template will not affect any
tenants created prior to the removal; the template will not be
copied into any future tenants.


**Request**

```sh
curl -H 'Accept: application/json' \
     -X DELETE https://shield.host/v2/global/policies/:uuid \
```

**Response**

```json
{
  "ok": "Retention policy template deleted successfully"
}
```

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

The following error messages can be returned:

- **Unable to retrieve retention policy template information**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such retention policy template**:
  No retention policy with the given UUID exists on
  the specified tenant.

- **Unable to delete retention policy template**:
  an internal error occurred and should be investigated by the
  site administrators

- **The retention policy template cannot be deleted at this time**:
  This retention policy is referenced by one or more
  extant job configuration; deleting it would lead to
  an incomplete (and unusable) setup.  Note that this
  error should never happen.


