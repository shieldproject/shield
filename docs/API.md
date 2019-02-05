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

    {
      "error": "No such retention policy"
    }

This format is used for all non-validation error reporting.

### Missing Values Format for Error Reporting

The **Missing Values Format** for error reporting is used for
reporting request validation errors where required fields in the
request payload are missing.  It consists of a top-level JSON
object containing a single key, `missing`, which is set to a list
of field names that must be sent in the request, but were not.

Example:

    {
      "missing": [
        "name",
        "endpoint",
        "agent"
      ]
    }

The order of the fields is inconsequential.
## Health & Informational

The health and informational endpoints give you a glimpse into the
well-being of a SHIELD Core, for monitoring purposes, at various
levels of detail.


### GET /v2/info

Returns minimal information necessary for forward-compatibility
with different versions of SHIELD Core and clients.


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/info


This endpoint takes no query string parameters.

**Response**

    {
      "version" : "6.7.2",
      "env"     : "PRODUCTION",
      "color"   : "yellow",
      "motd"    : "Welcome to S.H.I.E.L.D.",
      "ip"      : "10.0.0.5",
      "api"     : 2
    }

The `version` key only shows up if the request was made in the
context of an authenticated session.

The `env` key is configurable by the SHIELD site administrator,
at deployment / boot time.

Similar to env, the `color` key is configurable by the SHIELD site
administrator, used to visually differentiate various SHIELD
deployments in the WebUI.

The `motd` is what is displayed upon login, and can be changed by
the SHIELD site administrator.

The `ip` key is the ip of the SHIELD core.

The `api` key is a single integer that identifies which version
of the SHIELD API this core implements.  Currently, there are
two possible values:

  - **2** - The `/v2` endpoints are present, but not `/v1`
  - **1** - Only the `/v1` endpoints are present (legacy).

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

This API endpoint does not return any error conditions.

### GET /v2/bearings

Returns a baseline set of information about the data related to the
current session's view of SHIELD, including targets, jobs, stores for
all assigned tenants, and global storage.


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/bearings


This endpoint takes no query string parameters.

**Response**

    {
      "vault"  : "unlocked",
      "shield" : {
        "version" : "6.7.2",
        "env"     : "PRODUCTION",
        "color"   : "yellow",
        "motd"    : "Welcome to S.H.I.E.L.D.",
        "ip"      : "10.0.0.5",
        "api"     : 2
      },
    
      "user" : {
        "uuid"           : "229db66f-ad6b-40ee-b0c9-36b3eb957503",
        "name"           : "Persephone Jones",
        "account"        : "pjones",
        "backend"        : "local",
        "sysrole"        : "technician",
        "default_tenant" : "c76869df-3535-4a59-b33b-2f650e660bf1"
      },
    
      "stores": [
        {
          "uuid"    : "8f4dd8c4-9677-4660-ad53-b0a5b718628c",
          "name"    : "Global Storage",
          "summary" : "Global Storage, for use by any and all",
          "global"  : true,
    
          "agent"  : "127.0.0.1:5444",
          "plugin" : "webdav",
    
          "healthy"        : true,
          "archive_count"  : 0,
          "storage_used"   : 0,
          "threshold"      : 0,
          "daily_increase" : 0,
    
          "config": {
            "url": "http://localhost:8182"
          },
    
          "last_test_task_uuid": ""
        }
      ],
    
      "tenants" : {
        "c76869df-3535-4a59-b33b-2f650e660bf1" : {
          "tenant" : {
            "uuid"           : "c76869df-3535-4a59-b33b-2f650e660bf1",
            "name"           : "My Tenant",
            "archive_count"  : 0,
            "storage_used"   : 0,
            "daily_increase" : 0
          },
    
          "role"   : "admin",
          "grants" : {
            "admin"    : true,
            "engineer" : true,
            "operator" : true
          },
    
          "archives": [],
          "jobs": [
            {
              "uuid"      : "c623b8bf-b631-43ee-bb44-949454702716",
              "name"      : "Hourly",
              "summary"   : "",
    
              "keep_n"    : 48,
              "keep_days" : 2,
    
              "schedule"  : "hourly at :05",
              "paused"    : true,
    
              "agent"     : "127.0.0.1:5444",
              "fixed_key" : false,
    
              "healthy"          : false,
              "last_run"         : 1543328457,
              "last_task_status" : "canceled",
    
              "target" : {
                "uuid" : "5c180612-05e8-4ae6-9046-9d40d50d1a3c",
                "name" : "SHIELD",
    
                "agent"  : "",
                "plugin" : "fs",
    
                "compression" : "bzip2",
                "endpoint"    : "{\"base_dir\":\"/e/no/ent\",\"bsdtar\":\"bsdtar\",\"exclude\":\"var/*.db\"}"
              },
              "store" : {
                "uuid"     : "a5d64d9e-acc7-4621-8489-2ede2d3b31bf",
                "name"     : "CloudStor",
                "summary"  : "A temporary store for the dev environment.",
    
                "healthy"  : true,
    
                "agent"    : "",
                "plugin"   : "webdav",
                "endpoint" : "{\"url\":\"http://localhost:8182\"}"
              }
            }
          ],
    
          "targets" : [
            {
              "uuid"    : "5c180612-05e8-4ae6-9046-9d40d50d1a3c",
              "name"    : "SHIELD",
              "summary" : "The working directory of the dev environment.",
    
              "compression" : "bzip2",
    
              "agent"  : "127.0.0.1:5444",
              "plugin" : "fs",
              "config" : {
                "base_dir" : "/e/no/ent",
                "bsdtar"   : "bsdtar",
                "exclude"  : "var/*.db"
              }
            }
          ],
    
          "stores" : [
            {
              "uuid"    : "a5d64d9e-acc7-4621-8489-2ede2d3b31bf",
              "name"    : "CloudStor",
              "summary" : "A temporary store for the dev environment.",
              "global"  : false,
    
              "healthy"        : true,
              "archive_count"  : 0,
              "storage_used"   : 0,
              "daily_increase" : 0,
              "threshold"      : 0,
    
              "agent"   : "127.0.0.1:5444",
              "plugin"  : "webdav",
              "config"  : {
                "url": "http://localhost:8182"
              },
    
              "last_test_task_uuid": "4320b06d-dd3d-4c00-a2cd-9fec9b2520d6"
            }
          ]
        }
      }
    }

FIXME

**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

This API endpoint does not return any error conditions.

### GET /v2/health

Returns health information about the SHIELD Core, connected
storage accounts, and general metrics, at a global scope.


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/health


This endpoint takes no query string parameters.

**Response**

If all goes well, you will receive a 200 OK, with a `Content-Type`
of `application/json`, and something similar to the following JSON
payload in the response body:

    {
      "health": {
        "core"       : "unsealed",
        "storage_ok" : true,
        "jobs_ok"    : true
      },
      "storage": [
        { "name": "s3", "healthy": true },
        { "name": "fs", "healthy": true } ],
      "jobs": [
        {
          "uuid"    : "3dc875a4-042c-47a1-828c-1d927455c6c7",
          "target"  : "BOSH DB",
          "job"     : "daily",
          "healthy" : true
        },
        {
          "uuid"    : "d9a4547e-c1e7-4869-bb8d-4abb757b2f70",
          "target"  : "BOSH DB",
          "job"     : "weekly",
          "healthy" : true
        }
      ],
      "stats": {
        "jobs"    : 8,
        "systems" : 7,
        "archives": 124,
        "storage" : 243567112,
        "daily"   : 12345000
      }
    }

**Access Control**

You must be authenticated to access this API endpoint.

**Errors**

The following error messages can be returned:

- **Unable to check SHIELD health**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.


### GET /v2/tenants/:tenant/health

Returns health information about the SHIELD Core, connected
storage accounts, and general metrics, restricted to the scope
visible to a single tenant.


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/tenants/:tenant/health


This endpoint takes no query string parameters.

**Response**

If all goes well, you will receive a 200 OK, with a `Content-Type`
of `application/json`, and something similar to the following JSON
payload in the response body:

    {
      "health": {
        "core"       : "unsealed",
        "storage_ok" : true,
        "jobs_ok"    : true
      },
      "storage": [
        { "name": "s3", "healthy": true },
        { "name": "fs", "healthy": true } ],
      "jobs": [
        {
          "uuid"    : "3dc875a4-042c-47a1-828c-1d927455c6c7",
          "target"  : "BOSH DB",
          "job"     : "daily",
          "healthy" : true
        },
        {
          "uuid"    : "d9a4547e-c1e7-4869-bb8d-4abb757b2f70",
          "target"  : "BOSH DB",
          "job"     : "weekly",
          "healthy" : true
        }
      ],
      "stats": {
        "jobs"    : 8,
        "systems" : 7,
        "archives": 124,
        "storage" : 243567112,
        "daily"   : 12345000
      }
    }

**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `operator` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to check SHIELD health**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


## SHIELD Authentication

The Authentication endpoints allow clients to authenticate to a
SHIELD Core, providing credentials to prove their identity and
their authorization to perform other tasks inside of SHIELD.


### POST /v2/auth/login

Authenticate against the SHIELD API as a local user, and retrieve
a session ID that can be used for future, authenticated,
interactions.


**Request**

    curl -H 'Accept: application/json' \
         -H 'Content-Type: application/json' \
         -X POST https://shield.host/v2/auth/login \
         --data-binary '
    {
      "username": "your-username",
      "password": "your-password"
    }'


**NOTE:** `password` is sent in cleartext, so SHIELD should aways be
communicating over TLS (HTTPS).

Both fields, `username`, and `password`, are required.

**Response**

    {
      "ok": "95cca9ea-d2e6-4966-b071-9df6856a0e55"
    }

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

    curl -H 'Accept: application/json' \
            https://shield.host/v2/auth/logout


This endpoint takes no query string parameters.

**Response**

    {
      "ok" : "Successfully logged out"
    }

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

    curl -H 'Accept: application/json' \
            https://shield.host/v2/auth/id


This endpoint takes no query string parameters.

**Response**

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


### GET /v2/auth/providers

Retrieve public configuration of SHIELD Authentication Providers,
which can be used by client systems to initiate authentication
against 3rd party systems like Github and Cloud Foundry UAA.


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/auth/providers


This endpoint takes no query string parameters.

**Response**

    [
      {
        "name"       : "Github(.com)",
        "identifier" : "gh",
        "type"       : "github",
    
        "web_entry"  : "/auth/gh/web",
        "cli_entry"  : "/auth/gh/cli",
        "redirect"   : "/auth/gh/redir"
      }
    ]
**Access Control**

This endpoint requires no authentication or authorization.

**Errors**

This API endpoint does not return any error conditions.

### GET /v2/auth/providers/:identifier

Retrieve private configuration of a single SHIELD Authentication
Providers, for use by SHIELD administrators.  This includes all of
the information from the public endpoint, as well as private
properties including things like client secrets.


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/auth/providers/:identifier


This endpoint takes no query string parameters.

**Response**

    {
      "name"       : "Github(.com)",
      "identifier" : "gh",
      "type"       : "github",
    
      "web_entry"  : "/auth/gh/web",
      "cli_entry"  : "/auth/gh/cli",
      "redirect"   : "/auth/gh/redir",
    
      "properties" : {
        "secret"   : "properties",
        "that"     : "are",
        "provider" : "specific"
      }
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `admin` system role.

**Errors**

The following error messages can be returned:

- **No such authentication provider**:
  No authentication provider was found with the given
  identifier.

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### GET /v2/auth/tokens

Retrieve a list of all authentication tokens that have been
generated for use on behalf of the currently authenticated
user.  For security reasons, the session ID attached to the
authentication token is not returned.


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/auth/tokens


This endpoint takes no query string parameters.

**Response**

    [
      {
        "uuid"       : "bbcb6675-8ec4-4412-93e3-35626860b126",
        "name"       : "test",
        "created_at" : "2017-10-21 00:54:33",
        "last_seen"  : null
      }
    ]

The `uuid` key is used strictly for retrieving and revoking each
authentication token; it cannot be used as an authentication token,
as it is not the session ID.

**Access Control**

You must be authenticated to access this API endpoint.

**Errors**

The following error messages can be returned:

- **Authentication failed**:
  The request either lacked a session cookie (or an
  `X-Shield-Session` header), or some other internal
  error has occurred, and SHIELD administrators should
  investigate.

- **Unable to retrieve tokens information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.


### POST /v2/auth/tokens

Generate a new authentication token to act on behalf of the
currently authenticated user.


**Request**

    curl -H 'Accept: application/json' \
         -H 'Content-Type: application/json' \
         -X POST https://shield.host/v2/auth/tokens \
         --data-binary '
    {
      "name": "auth-token-name"
    }'


Each authentication token requires a name that is unique
to the parent user account.

**Response**

    {
      "uuid"       : "bbcb6675-8ec4-4412-93e3-35626860b126",
      "session"    : "8ef409e9-690d-4d91-9f74-6d657f56843e",
      "name"       : "test",
      "created_at" : "2017-10-21 00:54:33",
      "last_seen"  : null
    }

The `uuid` key is used strictly for retrieving and revoking each
authentication token; it cannot be used as an authentication token,
as it is not the session ID.

The `session` key should be used

**Access Control**

You must be authenticated to access this API endpoint.

**Errors**

The following error messages can be returned:

- **Authentication failed**:
  The request either lacked a session cookie (or an
  `X-Shield-Session` header), or some other internal
  error has occurred, and SHIELD administrators should
  investigate.

- **Unable to retrieve tokens information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unable to generate new token**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.


### DELETE /v2/auth/tokens/:uuid

Revoke an authentication token that belongs to the currently
authenticated user (even if its authenticated by the token being
revoked).


**Request**

    curl -H 'Accept: application/json' \
         -X DELETE https://shield.host/v2/auth/tokens/:uuid \


This endpoint takes no query string parameters.

**Response**

    {
      "ok": "Token revoked"
    }

Note that if you have revoked the auth token you were using, all
subsequent requests will fail to authenticated.  It makes sense,
but it bears repeating.

**Access Control**

You must be authenticated to access this API endpoint.

**Errors**

The following error messages can be returned:

- **Unable to revoke auth token**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.


### GET /v2/auth/sessions

Retrieve a list of all current login sessions


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/auth/sessions


- **?exact=(t|f)**
When filtering sessions, perform either exact field / value
matching (`exact=t`), or fuzzy search (`exact=f`, the
default)


- **?is_token=(t|f)**
When filtering sessions, include those associated with tokens (default - false)


- **?uuid=**
Only show the session that matches the given UUID.
This is a FIXME - we probably need to remove this.


- **?user_uuid=**
Only show sessions that matches the given user UUID.


- **?name=...**
Only show sessions whose associated token name match the given value.
Subject to the `exact=(t|f)` query string parameter.


- **?ip_addr=...**
Only show sessions who are associated with the given IP address.


- **?limit=N**
Limit the returned result set to the first _limit_ users
that match the other filtering rules.  A limit of `0` (the
default) denotes an unlimited search.



**Response**

    [
      {
        "uuid": "cbeffb8d-4d3d-49a1-b4cd-14b344dac1f2",
        "user_uuid": "ccc0430b-9d3d-4b1c-a980-dac769f64174",
        "created_at": "2017-10-24 16:39:03",
        "last_seen_at": "2017-10-24 16:39:03",
        "token_uuid": "",
        "name": "",
        "ip_addr": "127.0.0.1",
        "user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36",
        "user_account": "admin",
        "current_session": false
      }
    ]

- **uuid** - The internal UUID assigned to this session by the SHIELD Core.

- **user\_uuid** - The internal UUID corresponding to the user associated with the session.

- **created\_at** - When the user created the session (upon login via WebUI or CLI).
  Date is formatted YYYY-MM-DD HH:MM:SS, in 24-hour notation.

- **last\_seen\_at** - When the user last made contact with the
  SHIELD Core via an authenticated endpoint.
  Date is formatted YYYY-MM-DD HH:MM:SS, in 24-hour notation.

- **token\_uuid** - The uuid of the SHIELD Token associated with the session (if applicable).

- **name** - The name of the SHIELD Token associated with the session (if applicable).

- **ip\_addr** - The originating `IP address` of the user, from the
  point-of-view of the SHIELD Core.

- **user\_agent** - The user agent associated with the last request made by the user.

- **user\_account** - The account corresponding to the user associated with the session.

- **current\_session** - Denotes whether the session corresponds to the requesting session.

**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `admin` system role.

**Errors**

The following error messages can be returned:

- **Authentication failed**:
  The request either lacked a session cookie (or an
  `X-Shield-Session` header), or some other internal
  error has occurred, and SHIELD administrators should
  investigate.

- **Unable to retrieve session information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### GET /v2/auth/sessions/:uuid

Retrieve a single login session


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/auth/sessions/:uuid


This endpoint takes no query string parameters.

**Response**

    {
      "uuid": "cbeffb8d-4d3d-49a1-b4cd-14b344dac1f2",
      "user_uuid": "ccc0430b-9d3d-4b1c-a980-dac769f64174",
      "created_at": "2017-10-24 16:39:03",
      "last_seen_at": "2017-10-24 16:39:03",
      "token_uuid": "",
      "name": "",
      "ip_addr": "127.0.0.1",
      "user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36",
      "user_account": "admin"
    }

- **uuid** - The internal UUID assigned to this session by the SHIELD Core.

- **user\_uuid** - The internal UUID corresponding to the user associated with the session.

- **created\_at** - When the user created the session (upon login via WebUI or CLI).
  Date is formatted YYYY-MM-DD HH:MM:SS, in 24-hour notation.

- **last\_seen\_at** - When the user last made contact with the
  SHIELD Core via an authenticated endpoint.
  Date is formatted YYYY-MM-DD HH:MM:SS, in 24-hour notation.

- **token\_uuid** - The uuid of the SHIELD Token associated with the session (if applicable).

- **name** - The name of the SHIELD Token associated with the session (if applicable).

- **ip\_addr** - The originating `IP address` of the user, from the
  point-of-view of the SHIELD Core.

- **user\_agent** - The user agent associated with the last request made by the user.

- **user\_account** - The account corresponding to the user associated with the session.

**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `admin` system role.

**Errors**

The following error messages can be returned:

- **Authentication failed**:
  The request either lacked a session cookie (or an
  `X-Shield-Session` header), or some other internal
  error has occurred, and SHIELD administrators should
  investigate.

- **Unable to retrieve session information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### DELETE /v2/auth/sessions/:uuid

Revoke a user's session and force them to reauthenticate on next request.


**Request**

    curl -H 'Accept: application/json' \
         -X DELETE https://shield.host/v2/auth/sessions/:uuid \


This endpoint takes no query string parameters.

**Response**

    {
      "ok": "Successfully cleared session 'd3092979-5a83-4006-8819-fd1695f9041f' (127.0.0.1)"
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `admin` system role.

**Errors**

The following error messages can be returned:

- **Session not found**:
  The session given by the URL UUID was not found in the database

- **Unable to clear session**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unable to retrieve session information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### PATCH /v2/auth/user/settings

Save user settings.


**Request**

    curl -H 'Accept: application/json' \
         -H 'Content-Type: application/json' \
         -X PATCH https://shield.host/v2/auth/user/settings \
         --data-binary '
    {
      "default_tenant": "2a03d67b-6146-4716-b10a-42ec073cfb78"
    }'


This endpoint takes no query string parameters.

**Response**

    {
      "ok" : "Settings saved."
    }
**Access Control**

You must be authenticated to access this API endpoint.

**Errors**

The following error messages can be returned:

- **Authentication failed**:
  The request either lacked a session cookie (or an
  `X-Shield-Session` header), or some other internal
  error has occurred, and SHIELD administrators should
  investigate.

- **Unable to save settings**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.


## SHIELD Users

SHIELD provides both a local user database, and the ability to
map 3rd party systems into the SHIELD tenancy model.  These API
endpoints allow you to manage the former, and query the results
of the latter.


### GET /v2/auth/local/users




**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/auth/local/users


- **?exact=(t|f)**
When filtering users, perform either exact field / value
matching (`exact=t`), or fuzzy search (`exact=f`, the
default)


- **?uuid=**
Only show the local user that matches the given UUID.
This is a FIXME - we probably need to remove this.


- **?account=...**
Only show local users whose account names (usernames)
match the given value.  Subject to the `exact=(t|f)` query
string parameter.


- **?sysrole=...**
Only show local users who have been assigned the given
system role.


- **?limit=N**
Limit the returned result set to the first _limit_ users
that match the other filtering rules.  A limit of `0` (the
default) denotes an unlimited search.



**Response**

    [
      {
        "uuid"       : "b30bb2dd-81d4-407f-91eb-a82ed3023218",
        "name"       : "Full Name",
        "account"    : "username",
        "sysrole"    : "engineer",
    
        "tenants": [
          {
            "uuid" : "d1fb6abf-55f2-4901-9662-8c6339e0a7d7",
            "name" : "Tenant Name",
            "role" : "operator"
          }
        ]
      }
    ]
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `manager` system role.

**Errors**

The following error messages can be returned:

- **Invalid limit parameter given**:
  Request specified a `limit` parameter that was either
  non-numeric, or was negative.  Note that `0` is a valid limit,
  standing for _unlimited_.

- **Unable to retrieve local users information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### GET /v2/auth/local/users/:uuid




**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/auth/local/users/:uuid


This endpoint takes no query string parameters.

**Response**

    {
      "uuid"       : "b30bb2dd-81d4-407f-91eb-a82ed3023218",
      "name"       : "Full Name",
      "account"    : "username",
      "sysrole"    : "engineer",
    
      "tenants": [
        {
          "uuid" : "d1fb6abf-55f2-4901-9662-8c6339e0a7d7",
          "name" : "Tenant Name",
          "role" : "operator"
        }
      ]
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `manager` system role.

**Errors**

The following error messages can be returned:

- **Unable to retrieve local user information**:
  an internal error occurred and should be investigated by the
  site administrators

- **user '...' not found (for local auth provider)**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### POST /v2/auth/local/users




**Request**

    curl -H 'Accept: application/json' \
         -H 'Content-Type: application/json' \
         -X POST https://shield.host/v2/auth/local/users \
         --data-binary '
    {
      "name"     : "Full Name",
      "account"  : "username",
      "password" : "cleartext password",
      "sysrole"  : "engineer"
    }'


The `sysrole` parameter must be either empty (or not specified),
or one of the following values: **admin**, **engineer**, or
**operator**.

**Response**

    {
      "uuid"       : "b30bb2dd-81d4-407f-91eb-a82ed3023218",
      "name"       : "Full Name",
      "account"    : "username",
      "sysrole"    : "engineer"
    }

**NOTE** that the user's password (neither in hashed form, or in
the clear) is never returned to requesters.

**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `manager` system role.

**Errors**

The following error messages can be returned:

- **System role '...' is invalid**:
  Request specified a `sysrole` payload parameter that was
  outside of the allowable set of values.

- **Unable to create local user '...'**:
  an internal error occurred and should be investigated by the
  site administrators

- **User '...' already exists**:
  You attempted to create a new local user account, but the
  username was already assigned to a pre-existing account.
  The request should not be retried.

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### PATCH /v2/auth/local/users/:uuid




**Request**

    curl -H 'Accept: application/json' \
         -H 'Content-Type: application/json' \
         -X PATCH https://shield.host/v2/auth/local/users/:uuid \
         --data-binary '
    {
      "name"     : "Full Name",
      "password" : "cleartext password",
      "sysrole"  : "engineer"
    }'


The `sysrole` parameter must be either empty (or not specified),
or one of the following values: **admin**, **engineer**, or
**operator**.

**NOTE** that you cannot alter a users `account` after creation.

**Response**

    {
      "uuid"       : "b30bb2dd-81d4-407f-91eb-a82ed3023218",
      "name"       : "Full Name",
      "account"    : "username",
      "sysrole"    : "engineer"
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `manager` system role.

**Errors**

The following error messages can be returned:

- **No such local user**:
  Either the user given by the URL UUID was not found in the
  database, or that user was created by an authentication
  provider, and not SHIELD itself.

- **System role '...' is invalid**:
  Request specified a `sysrole` payload parameter that was
  outside of the allowable set of values.

- **Unable to update local user '...'**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### DELETE /v2/auth/local/users/:uuid




**Request**

    curl -H 'Accept: application/json' \
         -X DELETE https://shield.host/v2/auth/local/users/:uuid \


This endpoint takes no query string parameters.

**Response**

    {
      "ok" : "Successfully deleted local user"
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `manager` system role.

**Errors**

The following error messages can be returned:

- **Local User '...' not found**:
  Either the user given by the URL UUID was not found in the
  database, or that user was created by an authentication
  provider, and not SHIELD itself.

- **Unable to retrieve local user information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unable to delete local user '...'**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


## SHIELD Core

These endpoints allow clients to initialize brand new SHIELD
Cores, and unlock or rekey existing ones.


### POST /v2/init

Initializes a new SHIELD Core, to set up the encryption facilities
for storing backup archive encryption keys safely and securely.
Your SHIELD Core can only be initialized once.


**Request**

    curl -H 'Accept: application/json' \
         -H 'Content-Type: application/json' \
         -X POST https://shield.host/v2/init \
         --data-binary '
    {
      "master" : "your secret master password"
    }'


Where:

  - message: master is the plaintext master password
    to use for encrypting the credentials to the
    SHIELD Core storage vault.

**Response**

If all went well, and the SHIELD Core was properly initialized,
you will receive a 200 OK, and the following response:

    {
      "ok" : "Successfully initialized the SHIELD Core"
    }

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

    curl -H 'Accept: application/json' \
         -H 'Content-Type: application/json' \
         -X POST https://shield.host/v2/unlock \
         --data-binary '
    {
      "master" : "your secret master password"
    }'


- message: master is the plaintext master password
  that was created when you initialized this SHIELD
  Core (or whatever you last rekeyed it to be).

**Response**

On success, you will receive a 200 OK, with the
following response:

    {
      "ok" : "Successfully unlocked the SHIELD Core"
    }

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

    curl -H 'Accept: application/json' \
         -H 'Content-Type: application/json' \
         -X POST https://shield.host/v2/rekey \
         --data-binary '
    {
      "current" : "your CURRENT master password",
      "new"     : "what you want to change it to"
    }'


This endpoint takes no query string parameters.

**Response**

If all goes well, you will receive a 200 OK, and the
following response:

    {
      "ok" : "Successfully rekeyed the SHIELD core"
    }

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

    curl -H 'Accept: application/json' \
            https://shield.host/v2/agents


This endpoint takes no query string parameters.

**Response**

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

The top-level `agents` key is a list of object describing each registered agent:

- **name** - The name of the SHIELD Agent, as set by
  the local system administrator (which may not be the
  SHIELD site administrator).

- **uuid** - The internal UUID assigned to this agent by the SHIELD Core.

- **address** - The `host:port` of the agent, from the
  point-of-view of the SHIELD Core.

- **version** - The version of the remote SHIELD Agent's software.

- **status** - The health status of the remote SHIELD
  Agent, one of `ok` or `failing`.

- **hidden** - Whether or not this agent has been administratively hidden.

- **last\_error** - TBD

- **last\_seen\_at** - When the remote SHIELD Agent last
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

You must be authenticated to access this API endpoint.

You must also have the `admin` system role.

**Errors**

The following error messages can be returned:

- **Unable to retrieve agent information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


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

    curl -H 'Accept: application/json' \
         -H 'Content-Type: application/json' \
         -X POST https://shield.host/v2/agents \
         --data-binary '
    {
      "name" : "some-identifier",
      "port" : 5444
    }'


Where:

- message: name is the name of the agent to display in
  the backend, and in log messages.  Usually, an FQDN
  or other unique host identifier is preferable here.
- message: port is the port number that the SHIELD
  agent is bound to.  The remote peer IP will be
  determined from the HTTP request's peer address.

**Response**

On success, you will receive a 200 OK, and the following response:

    {
      "ok" : "Pre-registered agent <name> at <host>:<port>"
    }

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

    curl -H 'Accept: application/json' \
            https://shield.host/v2/agents/:uuid


This endpoint takes no query string parameters.

**Response**

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

You must be authenticated to access this API endpoint.

You must also have the `admin` system role.

**Errors**

The following error messages can be returned:

- **Unable to retrieve agent information**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such agent**:
  The requested agent UUID was not found in the list
  of registered agents.

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### POST /v2/agents/:uuid/show

Mark a single SHIELD agent as visible to all tenants, and
available for configuration.


**Request**

    curl -H 'Accept: application/json' \
         -X POST https://shield.host/v2/agents/:uuid/show \


This endpoint takes no query string parameters.

**Response**

    {
      "ok" : "Agent is now visible to everyone"
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `admin` system role.

**Errors**

The following error messages can be returned:

- **Unable to retrieve agent information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unable to set agent visibility**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such agent**:
  The requested agent UUID was not found in the list
  of registered agents.

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### POST /v2/agents/:uuid/hide

Hide a single SHIELD agent from all tenants, rendering it unusable
in storage and target configuration.  Pre-existing configurations
will continue to function; this only affects visibility for future
configuration.


**Request**

    curl -H 'Accept: application/json' \
         -X POST https://shield.host/v2/agents/:uuid/hide \


This endpoint takes no query string parameters.

**Response**

    {
      "ok" : "Agent is now visible to everyone"
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `admin` system role.

**Errors**

The following error messages can be returned:

- **Unable to retrieve agent information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unable to set agent visibility**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such agent**:
  The requested agent UUID was not found in the list
  of registered agents.

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### POST /v2/agents/:uuid/resync

Re-synchronize a SHIELD agent by immediately contacting it via the
agent channel and interrogating it for status and metadata.  This
allows administrators to force a re-check of an agent, regardless of
slow loop scheduling.


**Request**

    curl -H 'Accept: application/json' \
         -X POST https://shield.host/v2/agents/:uuid/resync \


This endpoint takes no query string parameters.

**Response**

    {
      "ok" : "Ad hoc agent resynchronization underway"
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `admin` system role.

**Errors**

The following error messages can be returned:

- **Unable to retrieve agent information**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such agent**:
  The requested agent UUID was not found in the list
  of registered agents.

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### GET /v2/tenants/:tenant/agents

Retrieves information about all registered SHIELD Agents,
viewable by a given tenant.


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/tenants/:tenant/agents


This endpoint takes no query string parameters.

**Response**

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
      ]
    }

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

**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `operator` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve agent information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### GET /v2/tenants/:tenant/agents/:uuid

Retrieve extended information about a single SHIELD Agent,
viewable by a given tenant.  This includes plugin metadata, but
not problem information (which is accessible via `GET /v2/agents`,
but only to SHIELD administrators).


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/tenants/:tenant/agents/:uuid


This endpoint takes no query string parameters.

**Response**

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
      }
    }

The top-level `agents` key contains the same agent
information that the `GET /v2/tenants/:tenant/agents` endpoint
returns.

The `metadata` key is exclusive to this endpoint, and
contains all of the agent metadata.  Of particular
interest is the `plugins` key, which contains a map of
plugin metadata, keyed by the plugin name.  The format
of this metadata is documented in
[/docs/plugins.md](/docs/plugins.md).

**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `operator` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve agent information**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such agent**:
  The requested agent UUID was not found in the list
  of registered agents visible to this tenant.

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


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

    curl -H 'Accept: application/json' \
            https://shield.host/v2/tenants


- **?exact=(t|f)**
When filtering tenants, perform either exact field / value
matching (`exact=t`), or fuzzy search (`exact=f`, the
default)


- **?uuid=**
Only show the tenant that matches the given UUID.
This is a FIXME - we probably need to remove this.


- **?name=...**
Only show tenant whose name matches the given value.
Subject to the `exact=(t|f)` query string parameter.


- **?limit=N**
Limit the returned result set to the first _limit_ users
that match the other filtering rules.  A limit of `0` (the
default) denotes an unlimited search.



**Response**

    [
      {
        "uuid": "f2ebbb9f-87f9-43e0-8515-dfce5d4d844c",
        "name": "A Tenant",
    
        "archive_count"  : 32,
        "storage_used"   : 140509184,
        "daily_increase" : 1520435
      },
      {
        "uuid": "4b6f6e2a-6ac6-443e-a910-aa412744165e",
        "name": "Some Other Tenant",
    
        "archive_count"  : 2,
        "storage_used"   : 268435456,
        "daily_increase" : 268435456
      }
    ]

For each tenant, SHIELD will also return a handful of tenant
metrics, including how many backup archives belong to the tenant
(`archive_count`), the total amount of cloud storage used, in
bytes (`storage_used`), and a linear-fit daily delta of storage
usage (`daily_increase`), also in bytes.

**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `manager` system role.

**Errors**

The following error messages can be returned:

- **Unable to retrieve tenants information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### POST /v2/tenants

Create a new tenant.


**Request**

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

    {
      "name": "A New Tenant",
      "uuid": "52d20ef4-f154-431e-a5bb-bb3a200976bb",
    
      "archive_count"  : 0,
      "storage_used"   : 0,
      "daily_increase" : 0
    }

**NOTE**: You cannot (for obvious reasons) set the
`archive_count`, `storage_used` and `daily_increase` fields when
you create a tenant.

**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `manager` system role.

**Errors**

The following error messages can be returned:

- **Tenant name 'system' is reserved**:
  You attempted to create a tenant named system
  (case-insensitive), which is not allowed.  SHIELD uses the
  tenant name "SYSTEM" for its own, internal purposes.

- **Unable to creeate new tenant**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unable to install template retention policies into new tenant
**:
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

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### GET /v2/tenants/:uuid

Request more detailed information about a single tenant.


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/tenants/:uuid


This endpoint takes no query string parameters.

**Response**

    {
      "name": "A Tenant",
      "uuid": "f2ebbb9f-87f9-43e0-8515-dfce5d4d844c",
    
      "archive_count"  : 32,
      "storage_used"   : 140509184,
      "daily_increase" : 1520435,
    
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

The `members` key will be absent if this tenant has no members.

**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `manager` system role.

**Errors**

The following error messages can be returned:

- **Unable to retrieve tenant information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unable to retrieve tenant memberships information**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such tenant**:
  The requested tenant UUID was not found in the database.

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### PATCH /v2/tenants/:uuid

Update a tenant with new attributes.


**Request**

    curl -H 'Accept: application/json' \
         -H 'Content-Type: application/json' \
         -X PATCH https://shield.host/v2/tenants/:uuid \
         --data-binary '
    {
      "name" : "A New Name"
    }'


**NOTE**: You cannot (for obvious reasons) set the
`archive_count`, `storage_used` and `daily_increase` fields when
you update a tenant.

**Response**

    {
      "name" : "A New Name",
      "uuid" : "adcfee48-8b43-4ba3-9438-e0da55b8e9df",
    
      "archive_count"  : 32,
      "storage_used"   : 140509184,
      "daily_increase" : 1520435
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `manager` system role.

**Errors**

The following error messages can be returned:

- **Unable to update tenant**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unable to retrieve tenant information**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such tenant**:
  The requested tenant UUID was not found in the database.

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### POST /v2/tenants/:uuid/invite

Invite one or more local users to a tenant.


**Request**

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


Even if you only need to invite a single user, you must specify a list of
user objects.

**Response**

    {
      "ok" : "Invitations sent"
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `manager` system role.

**Errors**

The following error messages can be returned:

- **Unable to retrieve tenant information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unable to update tenant memberships information**:
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

- **No such tenant**:
  The requested tenant UUID was not found in the database.

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### POST /v2/tenants/:uuid/banish

Remove a user from a tenant they currently belong to.


**Request**

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


This endpoint takes no query string parameters.

**Response**

    {
      "ok" : "Banishments served."
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `manager` system role.

**Errors**

The following error messages can be returned:

- **Unable to retrieve tenant information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unable to update tenant memberships information**:
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

- **No such tenant**:
  The requested tenant UUID was not found in the database.

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### DELETE /v2/tenants/:uuid

Remove a tenant.


**Request**

    curl -H 'Accept: application/json' \
         -X DELETE https://shield.host/v2/tenants/:uuid \


This endpoint takes no query string parameters.

**Response**

    {
      "ok": "Successfully deleted tenant"
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `manager` system role.

**Errors**

The following error messages can be returned:

- **Unable to retrieve tenant information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unable to delete tenant**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such tenant**:
  The requested tenant UUID was not found in the database.

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


## SHIELD Targets

Targets represent the data systems that SHIELD runs backup and
restore operations against as course of normal function.


### GET /v2/tenants/:tenant/targets

Retrieve all defined targets for a tenant.


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/tenants/:tenant/targets


- **?exact=(t|f)**
When filtering targets, perform either exact field / value
matching (`exact=t`), or fuzzy search (`exact=f`, the
default)


- **?unused=(t|f)**
When filtering targets, skip those that are unused (true) or used (false)


- **?name=...**
Only show targets whose name matches the given value.
Subject to the `exact=(t|f)` query string parameter.


- **?plugin=...**
Only show targets who are associated with the given plugin.
Subject to the `exact=(t|f)` query string parameter.


- **?limit=N**
Limit the returned result set to the first _limit_ users
that match the other filtering rules.  A limit of `0` (the
default) denotes an unlimited search.



**Response**

    [
      {
        "uuid"     : "b4400ee0-dce9-4277-9948-02a56ad51b17",
        "name"     : "Some Target",
        "summary"  : "The operator-supplied description of this target",
        "agent"    : "127.0.0.1:5444",
        "plugin"   : "fs",
    
        "config" : {
          "target"   : "specific",
          "settings" : "and configuration"
        }
      }
    ]
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `operator` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve targets information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### POST /v2/tenants/:tenant/targets

Create a new target in a tenant.


**Request**

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


Note: the `endpoint` key is currently a string of JSON, which
means that it contains lots of escape sequences.  Future versions
of the v2 API (prior to launch) may alter this to be the full
JSON, inline, for both readability and sanity's sake.

FIXME: Fix target.endpoint string -> JSON problem.

The following query string parameters are honored:

- **?test=(t|f)**
Perform all validation and preparatory steps, but don't
actually create the target in the database.  This is useful
for validating that a target _could_ be created, without
creating it (i.e. for defering creation until later).




**Response**

    {
      "uuid"     : "b6d03df5-6978-43d8-ad9e-a22f8ec8457a",
      "name"     : "New Target Name",
      "summary"  : "A longer description of the target",
      "agent"    : "127.0.0.1:5444",
      "endpoint" : "{}",
      "plugin"   : "plugin"
    }

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

- **Unable to retrieve tenant information**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such tenant**:
  No tenant was found with the given UUID.
- **Unable to create new data target**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### GET /v2/tenants/:tenant/targets/:uuid

Retrieve a single target for a tenant.


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/tenants/:tenant/targets/:uuid


This endpoint takes no query string parameters.

**Response**

    {
      "uuid"     : "b4400ee0-dce9-4277-9948-02a56ad51b17",
      "name"     : "Some Target",
      "summary"  : "The operator-supplied description of this target",
      "agent"    : "127.0.0.1:5444",
      "endpoint" : "{}",
      "plugin"   : "fs"
    }

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

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### PUT /v2/tenants/:tenant/targets/:uuid

Update an existing target on a tenant.


**Request**

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


You can specify as many or few of these fields as you want;
omitted fields will be left at their previous values.

Note: the `endpoint` key is currently a string of JSON, which
means that it contains lots of escape sequences.  Future versions
of the v2 API (prior to launch) may alter this to be the full
JSON, inline, for both readability and sanity's sake.

FIXME: Fix target.endpoint string -> JSON problem.

**Response**

    {
      "ok" : "Updated target successfully"
    }

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

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### DELETE /v2/tenants/:tenant/targets/:uuid

Remove a target from a tenant.


**Request**

    curl -H 'Accept: application/json' \
         -X DELETE https://shield.host/v2/tenants/:tenant/targets/:uuid \


This endpoint takes no query string parameters.

**Response**

    {
      "ok": "Target deleted successfully"
    }

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

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


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

    curl -H 'Accept: application/json' \
            https://shield.host/v2/tenants/:tenant/stores


- **?exact=(t|f)**
When filtering stores, perform either exact field / value
matching (`exact=t`), or fuzzy search (`exact=f`, the
default)


- **?unused=(t|f)**
When filtering stores, skip those that are unused (true) or used (false)


- **?name=...**
Only show stores whose name matches the given value.
Subject to the `exact=(t|f)` query string parameter.


- **?plugin=...**
Only show stores who are associated with the given plugin.
Subject to the `exact=(t|f)` query string parameter.


- **?limit=N**
Limit the returned result set to the first _limit_ users
that match the other filtering rules.  A limit of `0` (the
default) denotes an unlimited search.



**Response**

    [
      {
        "uuid"    : "925c83ad-22e6-4cdd-bf63-6dd6d09cd86f",
        "name"    : "Cloud Storage Name",
        "global"  : false,
        "summary" : "A longer description of the storage configuration",
        "agent"   : "127.0.0.1:5444",
        "plugin"  : "fs",
        "config"  : {
          "base_dir" : "/var/data/root",
          "bsdtar"   : "bsdtar"
        },
        "threshold": 1073741824
      }
    ]

The values under `config` will depend entirely on what the
operator specified when they initially configured the storage
system.

**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `operator` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve storage systems information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### POST /v2/tenants/:tenant/stores

Create a new store on a tenant.


**Request**

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
      },
      "threshold": 1073741824
    }'


The values under `config` will depend entirely on which `plugin`
has been selected; no validation will be done by the SHIELD Core,
until the storage system is used in a job.

The following query string parameters are honored:

- **?test=(t|f)**
Perform all validation and preparatory steps, but don't
actually create the store in the database.  This is useful
for validating that a store _could_ be created, without
creating it (i.e. for defering creation until later).




**Response**

    {
      "name"    : "Storage System Name",
      "summary" : "A longer description for this storage system.",
      "plugin"  : "plugin-name",
      "agent"   : "127.0.0.1:5444",
      "config"  : {
        "plugin-specific": "configuration"
      },
      "threshold": 1073741824
    }

**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `engineer` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve storage system information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unable to create new storage system**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### GET /v2/tenants/:tenant/stores/:uuid

Retrieve a single store for the given tenant.


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/tenants/:tenant/stores/:uuid


This endpoint takes no query string parameters.

**Response**

    {
      "uuid"    : "925c83ad-22e6-4cdd-bf63-6dd6d09cd86f",
      "name"    : "Cloud Storage Name",
      "global"  : false,
      "summary" : "A longer description of the storage configuration",
      "plugin"  : "fs",
      "agent"   : "127.0.0.1:5444",
      "config"  : {
        "base_dir" : "/var/data/root",
        "bsdtar"   : "bsdtar"
      },
      "threshold": 1073741824
    }

The values under `config` will depend entirely on what the
operator specified when they initially configured the storage
system.

**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `operator` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve storage system information**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such storage system**:
  No storage system with the given UUID exists on the
  specified tenant.

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### PUT /v2/tenants/:tenant/stores/:uuid

Update an existing store on a tenant.


**Request**

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
      },
      "threshold": 1073741824
    }'


You can specify as many or few of these fields as you want;
omitted fields will be left at their previous values.  If `config`
is supplied, it will overwrite the value currently in the
database.

The values under `config` will depend entirely on which `plugin`
has been selected; no validation will be done by the SHIELD Core,
until the storage system is used in a job.

**Response**

    {
      "name"    : "Updated Store Name",
      "summary" : "A longer description of the storage system",
      "agent"   : "127.0.0.1:5444",
      "plugin"  : "plugin",
      "config"  : {
        "new": "plugin configuration"
      },
      "threshold": 1073741824
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `engineer` role on the tenant.

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

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### DELETE /v2/tenants/:tenant/stores/:uuid

Remove a store from a tenant.


**Request**

    curl -H 'Accept: application/json' \
         -X DELETE https://shield.host/v2/tenants/:tenant/stores/:uuid \


This endpoint takes no query string parameters.

**Response**

    {
      "ok": "Storage system deleted successfully"
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `engineer` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve storage system information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unable to delete storage system**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such storage system**:
  No storage system with the given UUID exists on the
  specified tenant.

- **The storage system cannot be deleted at this time**:
  This storage system is referenced by one or more
  extant job configuration; deleting it would lead to
  an incomplete (and unusable) setup.

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


## SHIELD Retention Policies

Retention Policies govern how long backup archives are kept,
to ensure that storage usage doesn't continue to increase
inexorably.


### GET /v2/tenants/:tenant/policies

Retrieve all defined retention policies for a tenant.


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/tenants/:tenant/policies


- **?exact=(t|f)**
When filtering policies, perform either exact field / value
matching (`exact=t`), or fuzzy search (`exact=f`, the
default)


- **?unused=(t|f)**
When filtering policies, skip those that are unused (true) or used (false)


- **?name=...**
Only show policies whose name matches the given value.
Subject to the `exact=(t|f)` query string parameter.


- **?limit=N**
Limit the returned result set to the first _limit_ users
that match the other filtering rules.  A limit of `0` (the
default) denotes an unlimited search.



**Response**

    [
      {
        "uuid"    : "f4dedf80-cdb2-4c81-9a58-b3a8282e3202",
        "name"    : "Long-Term Storage",
        "summary" : "A long-term solution, for infrequent backups only.",
        "expires" : 7776000
      }
    ]

The `expires` key is specified in seconds, but must
always be a multiple of 86400 (1 day).

**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `operator` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve retention policies information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### POST /v2/tenants/:tenant/policies

Create a new retention policy in a tenant.


**Request**

    curl -H 'Accept: application/json' \
         -H 'Content-Type: application/json' \
         -X POST https://shield.host/v2/tenants/:tenant/policies \
         --data-binary '
    {
      "name"    : "Retention Policy Name",
      "summary" : "A longer description of the policy",
      "expires" : 86400
    }'


The `expires` value must be specified in seconds, and
must be at least 86,400 (1 day) and be a multiple of
86,400.

The following query string parameters are honored:

- **?test=(t|f)**
Perform all validation and preparatory steps, but don't
actually create the policy in the database.  This is useful
for validating that a policy _could_ be created, without
creating it (i.e. for defering creation until later).




**Response**

    {
      "uuid"    : "4882b332-6182-4123-984f-f9e5dd8dae20",
      "name"    : "Retention Policy Name",
      "summary" : "A longer description of the policy",
      "expires" : 86400
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `engineer` role on the tenant.

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

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### GET /v2/tenants/:tenant/policies/:uuid

Retrieve a single retention policy for a tenant.


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/tenants/:tenant/policies/:uuid


This endpoint takes no query string parameters.

**Response**

    {
      "uuid"    : "4882b332-6182-4123-984f-f9e5dd8dae20",
      "name"    : "Retention Policy Name",
      "summary" : "A longer description of the policy",
      "expires" : 86400
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `operator` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve retention policy information**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such retention policy**:
  No retention policy with the given UUID exists on
  the specified tenant.

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### PUT /v2/tenants/:tenant/policies/:uuid

Update a single retention policy on a tenant.


**Request**

    curl -H 'Accept: application/json' \
         -H 'Content-Type: application/json' \
         -X PUT https://shield.host/v2/tenants/:tenant/policies/:uuid \
         --data-binary '
    {
      "name"    : "Updated Retention Policy Name",
      "summary" : "A longer description of the retention policy",
      "expires" : 86400
    }'


You can specify as many or few of these fields as you want;
omitted fields will be left at their previous values.

**Response**

    {
      "uuid"    : "4882b332-6182-4123-984f-f9e5dd8dae20",
      "name"    : "Retention Policy Name",
      "summary" : "A longer description of the policy",
      "expires" : 86400
    }

**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `engineer` role on the tenant.

**Errors**

The following error messages can be returned:

- **Retention policy expiry must be greater than 1 day**:
  You supplied an `expires` value less than 86,400.
  Please re-try the request with a higher value.

- **Retention policy expire must be a multiple of 1 day**:
  You supplied an `expires` value that was not a
  multiple of 86,400.  Please re-try the request with
  a different value.

- **Unable to retrieve retention policy template information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unable to update retention policy**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such retention policy**:
  No retention policy with the given UUID exists on
  the specified tenant.

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### DELETE /v2/tenants/:tenant/policies/:uuid

Remove a retention policy from a tenant.


**Request**

    curl -H 'Accept: application/json' \
         -X DELETE https://shield.host/v2/tenants/:tenant/policies/:uuid \


This endpoint takes no query string parameters.

**Response**

    {
      "ok": "Retention policy deleted successfully"
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `engineer` role on the tenant.

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

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


## SHIELD Jobs

Jobs are discrete, schedulable backup operations that tie together a
target system, a cloud storage system, a schedule, and a retention
policies.  Without Jobs, SHIELD cannot actually back anything up.


### GET /v2/tenants/:tenant/jobs

Retrieve all defined jobs for a tenant.


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/tenants/:tenant/jobs?name=redis&paused=t


By default, all jobs for the tenant will be returned.  You can
filter down to a subset of that by the following query string
parameters:

- **?exact=(t|f)**
When filtering jobs, perform either exact field / value
matching (`exact=t`), or fuzzy search (`exact=f`, the
default)


- **?paused=(t|f)**
Show only paused (`paused=t`) or unpaused (`paused=f`) jobs.
If omitted, all jobs are shown, regardless of their
pausedness.


- **?name=...**
Show only jobs whose names match the given value.
Subject to the `exact=(t|f)` query string parameter.


- **?target=...**
Show only jobs whose target UUIDs match the given value.
Subject to the `exact=(t|f)` query string parameter.


- **?store=...**
Show only jobs whose store UUIDs match the given value.
Subject to the `exact=(t|f)` query string parameter.


- **?policy=...**
Show only jobs whose retention policy UUIDs match the given
value.  Subject to the `exact=(t|f)` query string parameter.




**Response**

    {
      "uuid"        : "30f34d8f-762e-402a-b7ce-769a4a68de90",
      "name"        : "Job Name",
      "summary"     : "A longer description",
      "compression" : "bzip2",
      "expiry"      : 604800,
      "schedule"    : "daily 4am",
      "paused"      : false,
      "agent"       : "10.0.0.5:5444",
      "last_run"    : "2017-10-19 03:00:00",
      "status"      : "done",
    
      "policy" : {
        "uuid"    : "9a112894-10eb-439f-afd5-01597d8faf64",
        "name"    : "Retention Policy Name",
        "summary" : "A longer description"
      },
    
      "store" : {
        "uuid"    : "5945ef33-2cb6-4d7e-a9b7-43cce1773457",
        "name"    : "Cloud Storage System Name",
        "summary" : "A longer description",
        "plugin"  : "s3",
        "config"  : {
          "storage" : "configuration"
        }
      },
    
      "target" : {
        "uuid"   : "9f602367-256a-4454-be56-9ddde1257f13",
        "name"   : "Data System Name",
        "plugin" : "fs",
        "config" : {
          "target" : "configuration"
        }
      }
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `operator` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve tenant job information.**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### POST /v2/tenants/:tenant/jobs

Configure a new backup job on a tenant.


**Request**

    curl -H 'Accept: application/json' \
         -H 'Content-Type: application/json' \
         -X POST https://shield.host/v2/tenants/:tenant/jobs \
         --data-binary '
    {
      "name"        : "New Job Name",
      "summary"     : "A longer description...",
      "schedule"    : "daily 4am",
      "compression" : "bzip2",
      "paused"      : false,
    
      "store"       : "af1ad037-c8c1-4036-984a-3cf726b4081d",
      "target"      : "2c64d9ff-fc9f-4114-8e89-9f7c84fcaac7",
      "policy"      : "cb6b0503-4741-4cfd-9a1d-11b5a5aaadde"
    }'


**NOTE**: As of right now, the `store`, `target`, and `policy`
values must be passed as the UUIDs of the related objects.

FIXME : allow non-UUIDs for all three.

**Response**

    {
      "uuid"        : "30f34d8f-762e-402a-b7ce-769a4a68de90",
      "name"        : "Job Name",
      "summary"     : "A longer description",
      "compression" : "bzip2",
      "expiry"      : 604800,
      "schedule"    : "daily 4am",
      "paused"      : false,
      "agent"       : "10.0.0.5:5444",
    
      "last_run"         : "2017-10-19 03:00:00",
      "last_task_status" : "",
    
      "policy" : {
        "uuid"    : "9a112894-10eb-439f-afd5-01597d8faf64",
        "name"    : "Retention Policy Name",
        "summary" : "A longer description"
      },
    
      "store" : {
        "uuid"    : "5945ef33-2cb6-4d7e-a9b7-43cce1773457",
        "name"    : "Cloud Storage System Name",
        "summary" : "A longer description",
        "plugin"  : "s3",
        "config"  : {
          "storage" : "configuration"
        }
      },
    
      "target" : {
        "uuid"   : "9f602367-256a-4454-be56-9ddde1257f13",
        "name"   : "Data System Name",
        "plugin" : "fs",
        "config" : {
          "target" : "configuration"
        }
      }
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `engineer` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to create new job**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### GET /v2/tenants/:tenant/jobs/:uuid

Retrieve a single job for a tenant.


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/tenants/:tenant/jobs/:uuid


This endpoint takes no query string parameters.

**Response**

    {
      "uuid"        : "30f34d8f-762e-402a-b7ce-769a4a68de90",
      "name"        : "Job Name",
      "summary"     : "A longer description",
      "compression" : "bzip2",
      "expiry"      : 604800,
      "schedule"    : "daily 4am",
      "paused"      : false,
      "agent"       : "10.0.0.5:5444",
    
      "last_run"         : "2017-10-19 03:00:00",
      "last_task_status" : "",
    
      "policy" : {
        "uuid"    : "9a112894-10eb-439f-afd5-01597d8faf64",
        "name"    : "Retention Policy Name",
        "summary" : "A longer description"
      },
    
      "store" : {
        "uuid"    : "5945ef33-2cb6-4d7e-a9b7-43cce1773457",
        "name"    : "Cloud Storage System Name",
        "summary" : "A longer description",
        "plugin"  : "s3",
        "config"  : {
          "storage" : "configuration"
        }
      },
    
      "target" : {
        "uuid"   : "9f602367-256a-4454-be56-9ddde1257f13",
        "name"   : "Data System Name",
        "plugin" : "fs",
        "config" : {
          "target" : "configuration"
        }
      }
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `operator` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve tenant job information.**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such job**:
  The requested job was not found in the database, or
  it was not associated with the given tenant.

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### PUT /v2/tenants/:tenant/jobs/:uuid

Update a single job on a tenant.


**Request**

    curl -H 'Accept: application/json' \
         -H 'Content-Type: application/json' \
         -X PUT https://shield.host/v2/tenants/:tenant/jobs/:uuid \
         --data-binary '
    {
      "name"        : "New Name",
      "summary"     : "An updated summary",
      "compression" : "bzip2",
      "schedule"    : "daily 4am",
    
      "store"  : "a6ef5aea-51f6-4e91-a490-3063395f879b",
      "target" : "af1425ed-53fd-4ab6-a425-fb230c383901",
      "policy" : "c16a4783-19b8-400d-8b51-f47dcdc11da3"
    }'


Any of the fields in the request payload can be omitted to keep
the pre-existing value.

**NOTE**: As of right now, the `store`, `target`, and `policy`
values must be passed as the UUIDs of the related objects.

FIXME : allow non-UUIDs for all three.

**Response**

    {
      "ok": "Updated job successfully"
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `engineer` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve job information.**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such job**:
  The requested job was not found in the database, or
  it was not associated with the given tenant.

- **Unable to update job.**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### DELETE /v2/tenants/:tenant/jobs/:uuid

Remove a job from a tenant.


**Request**

    curl -H 'Accept: application/json' \
         -X DELETE https://shield.host/v2/tenants/:tenant/jobs/:uuid \


This endpoint takes no query string parameters.

**Response**

    {
      "ok": "Job deleted successfully"
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `engineer` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve job information.**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such job**:
  The requested job was not found in the database, or
  it was not associated with the given tenant.

- **Unable to delete job.**:
  an internal error occurred and should be investigated by the
  site administrators

- **The job could not be deleted at this time.**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### POST /v2/tenants/:tenant/jobs/:uuid/run

Perform an ad hoc backup job run


**Request**

    curl -H 'Accept: application/json' \
         -X POST https://shield.host/v2/tenants/:tenant/jobs/:uuid/run \


This endpoint takes no query string parameters.

**Response**

    {
      "ok": "Scheduled ad hoc backup job run",
      "task_uuid": "6d38e8bd-42e9-4c23-bbb6-9a480e0e2a82"
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `operator` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve job information.**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such job**:
  The requested job was not found in the database, or
  it was not associated with the given tenant.

- **Unable to schedule ad hoc backup job run.**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### POST /v2/tenants/:tenant/jobs/:uuid/pause

Pause a job, to prevent it from being scheduled.


**Request**

    curl -H 'Accept: application/json' \
         -X POST https://shield.host/v2/tenants/:tenant/jobs/:uuid/pause \


This endpoint takes no query string parameters.

**Response**

    {
      "ok": "Paused job successfully"
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `operator` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve job information.**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such job**:
  The requested job was not found in the database, or
  it was not associated with the given tenant.

- **Unable to pause job.**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### POST /v2/tenants/:tenant/jobs/:uuid/unpause

Unpause a job, allowing it to be scheduled again.


**Request**

    curl -H 'Accept: application/json' \
         -X POST https://shield.host/v2/tenants/:tenant/jobs/:uuid/unpause \


This endpoint takes no query string parameters.

**Response**

    {
      "ok": "Unpaused job successfully"
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `operator` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve job information.**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such job**:
  The requested job was not found in the database, or
  it was not associated with the given tenant.

- **Unable to unpause job.**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


## SHIELD Tasks

Tasks represent the context, status, and output of the execution of
some operation, be it a scheduled backup job being run, an ad hoc
restore operation, etc.

Since tasks represent internal state, they cannot easily be created or
updated via operators.  Instead, the execution of the job, or the
triggering of some other action, will cause a task to spring into
existence and move through its lifecycle.


### GET /v2/tenants/:tenant/tasks

Retrieve all tasks for a tenant.


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/tenants/:tenant/tasks


- **?active=(t|f)**
When filtering tasks, show those that are active (true) or inactive (false)


- **?status=...**
Only show tasks whose status matches the given value.


- **?target=**
Only show tasks associated with a given target...


- **?limit=N**
Limit the returned result set to the first _limit_ users
that match the other filtering rules.  A limit of `0` (the
default) denotes an unlimited search.



**Response**

    [
      {
        "uuid"         : "df2fd352-83b8-45b8-8f7b-ef74cf9eafdc",
        "owner"        : "user@backend",
        "type"         : "backup",
        "job_uuid"     : "bd2c5ac0-b499-4085-857e-69fa38441419",
        "archive_uuid" : "eb096379-7c45-4679-9b47-c563276dc22e",
        "status"       : "running",
        "started_at"   : "2017-10-17 04:51:16",
        "stopped_at"   : "",
        "log"          : "running log...",
        "notes"        : "Annotated notes about this task",
        "clear"        : "manual"
      }
    ]
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `operator` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve task information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Invalid limit parameter given**:
  Request specified a `limit` parameter that was either
  non-numeric, or was negative.  Note that `0` is a valid limit,
  standing for _unlimited_.

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### GET /v2/tenants/:tenant/tasks/:uuid

Retrieve a single task.


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/tenants/:tenant/tasks/:uuid


This endpoint takes no query string parameters.

**Response**

    {
      "uuid"         : "df2fd352-83b8-45b8-8f7b-ef74cf9eafdc",
      "owner"        : "user@backend",
      "type"         : "backup",
      "job_uuid"     : "bd2c5ac0-b499-4085-857e-69fa38441419",
      "archive_uuid" : "eb096379-7c45-4679-9b47-c563276dc22e",
      "status"       : "running",
      "started_at"   : "2017-10-17 04:51:16",
      "stopped_at"   : "",
      "log"          : "running log...",
      "notes"        : "Annotated notes about this task",
      "clear"        : "manual"
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `operator` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve task information**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such task**:
  You requested details on a task that either doesn't exist, or
  is not tied to the given tenant.

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### DELETE /v2/tenants/:tenant/tasks/:uuid

Cancel a task.


**Request**

    curl -H 'Accept: application/json' \
         -X DELETE https://shield.host/v2/tenants/:tenant/tasks/:uuid \


This endpoint takes no query string parameters.

**Response**

    {
      "ok": "Task canceled successfully"
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `operator` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve task information**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such task**:
  You requested details on a task that either doesn't exist, or
  is not tied to the given tenant.

- **Unable to cancel task**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


## SHIELD Backup Archives

Each backup archive contains the complete set of data retrieved from a
target system during a single backup job task.

Archives cannot be created _a priori_ via the API alone.  New archives
will be created from scheduled and ad hoc backup jobs, when they
succeed.


### GET /v2/tenants/:tenant/archives

Retrieve all archives for a tenant.


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/tenants/:tenant/archives


- **?target=**
Only show the archives that beloing to the the given target UUID.


- **?store=**
Only show the archives that beloing to the the given store UUID.


- **?before=**
Only show the archives created before the given time.


- **?after=**
Only show the archives created after the given time.


- **?status=...**
Only show the archives with the given status.


- **?limit=N**
Limit the returned result set to the first _limit_ users
that match the other filtering rules.  A limit of `0` (the
default) denotes an unlimited search.



**Response**

    [
      {
        "uuid"         : "5c8cef06-190c-4b07-a0b7-8452f6faff26",
        "key"          : "2017/10/27/2017-10-27-120512-948428f4-3a83-4b5d-8a41-7d27ca81ce8d",
        "taken_at"     : "2017-10-27 16:05:25",
        "expires_at"   : "2017-10-28 16:05:25",
        "notes"        : "",
        "compression"  : "bzip2",
        "encryption_type" : "aes256-ctr",
        "size"         : 43306681,
        "status"       : "valid",
        "purge_reason" : "",
        "job"          : "Hourly",
    
        "tenant_uuid" : "5524167e-cf56-4a8f-9580-cfca40949316",
    
        "target_uuid"     : "51d9cced-b11d-4b76-b9f3-fe0be4cd6087",
        "target_name"     : "SHIELD",
        "target_plugin"   : "fs",
        "target_endpoint" : "{\"base_dir\":\"/e/no/ent\",\"bsdtar\":\"bsdtar\",\"exclude\":\"var/*.db\"}",
    
        "store_uuid"     : "828fccae-a11e-41ee-bc13-d33c4dff1241",
        "store_name"     : "CloudStor",
        "store_plugin"   : "fs",
        "store_endpoint" : "{\"base_dir\":\"/tmp/shield.testdev.storeNy0fewQ\",\"bsdtar\":\"bsdtar\"}"
      }
    ]
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `operator` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve backup archives information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Invalid limit parameter given**:
  Request specified a `limit` parameter that was either
  non-numeric, or was negative.  Note that `0` is a valid limit,
  standing for _unlimited_.

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### GET /v2/tenants/:tenant/archives/:uuid

Retrieve a single archive for a tenant.


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/tenants/:tenant/archives/:uuid


This endpoint takes no query string parameters.

**Response**

    {
      "uuid"         : "5c8cef06-190c-4b07-a0b7-8452f6faff26",
      "key"          : "2017/10/27/2017-10-27-120512-948428f4-3a83-4b5d-8a41-7d27ca81ce8d",
      "taken_at"     : "2017-10-27 16:05:25",
      "expires_at"   : "2017-10-28 16:05:25",
      "notes"        : "",
      "compression"  : "bzip2",
      "encryption_type" : "aes256-ctr",
      "size"         : 43306681,
      "status"       : "valid",
      "purge_reason" : "",
      "job"          : "Hourly",
    
      "tenant_uuid" : "5524167e-cf56-4a8f-9580-cfca40949316",
    
      "target_uuid"     : "51d9cced-b11d-4b76-b9f3-fe0be4cd6087",
      "target_name"     : "SHIELD",
      "target_plugin"   : "fs",
      "target_endpoint" : "{\"base_dir\":\"/e/no/ent\",\"bsdtar\":\"bsdtar\",\"exclude\":\"var/*.db\"}",
    
      "store_uuid"     : "828fccae-a11e-41ee-bc13-d33c4dff1241",
      "store_name"     : "CloudStor",
      "store_plugin"   : "fs",
      "store_endpoint" : "{\"base_dir\":\"/tmp/shield.testdev.storeNy0fewQ\",\"bsdtar\":\"bsdtar\"}"
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `operator` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve backup archive information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Archive Not Found**:
  You requested details on an archive that either doesn't exist, or
  is not tied to the given tenant.

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### PUT /v2/tenants/:tenant/archives/:uuid

Update a single archive on a tenant.


**Request**

    curl -H 'Accept: application/json' \
         -H 'Content-Type: application/json' \
         -X PUT https://shield.host/v2/tenants/:tenant/archives/:uuid \
         --data-binary '
    {
      "notes": "Notes for this specific archive"
    }'


This endpoint takes no query string parameters.

**Response**

    {
      "uuid"         : "5c8cef06-190c-4b07-a0b7-8452f6faff26",
      "key"          : "2017/10/27/2017-10-27-120512-948428f4-3a83-4b5d-8a41-7d27ca81ce8d",
      "taken_at"     : "2017-10-27 16:05:25",
      "expires_at"   : "2017-10-28 16:05:25",
      "notes"        : "",
      "compression"  : "bzip2",
      "encryption_type" : "aes256-ctr",
      "size"         : 43306681,
      "status"       : "valid",
      "purge_reason" : "",
      "job"          : "Hourly",
    
      "tenant_uuid" : "5524167e-cf56-4a8f-9580-cfca40949316",
    
      "target_uuid"     : "51d9cced-b11d-4b76-b9f3-fe0be4cd6087",
      "target_name"     : "SHIELD",
      "target_plugin"   : "fs",
      "target_endpoint" : "{\"base_dir\":\"/e/no/ent\",\"bsdtar\":\"bsdtar\",\"exclude\":\"var/*.db\"}",
    
      "store_uuid"     : "828fccae-a11e-41ee-bc13-d33c4dff1241",
      "store_name"     : "CloudStor",
      "store_plugin"   : "fs",
      "store_endpoint" : "{\"base_dir\":\"/tmp/shield.testdev.storeNy0fewQ\",\"bsdtar\":\"bsdtar\"}"
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `operator` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve backup archive information**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such backup archive**:
  You requested details on an archive that either doesn't exist, or
  is not tied to the given tenant.

- **Unable to update backup archive**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### POST /v2/tenants/:tenant/archives/:uuid/restore

Restore a backup archive for a tenant, either to the original
target system it was taken from, or against a different target
owned by the same tenant.


**Request**

    curl -H 'Accept: application/json' \
         -H 'Content-Type: application/json' \
         -X POST https://shield.host/v2/tenants/:tenant/archives/:uuid/restore \
         --data-binary '
    {
      "target" : "2a1731c0-7d0e-4f31-8860-7d0ae7a34261"
    }'


Here, `target` is an optional UUID of an alternative data system
to restore this archive to.  This target must exist inside the
same tenant.  By default, if `target` is omitted, the archive
will be restored back to the target it was created from.

**Response**

When a restore has been scheduled for execution, it generates a
task inside of SHIELD, which is then returned as the result to
the requester:

    {
      "uuid"         : "df2fd352-83b8-45b8-8f7b-ef74cf9eafdc",
      "owner"        : "user@backend",
      "type"         : "restore",
      "job_uuid"     : "bd2c5ac0-b499-4085-857e-69fa38441419",
      "archive_uuid" : "eb096379-7c45-4679-9b47-c563276dc22e",
      "status"       : "running",
      "started_at"   : "2017-10-17 04:51:16",
      "stopped_at"   : "",
      "log"          : "running log...",
      "notes"        : "Annotated notes about this task",
      "clear"        : "manual"
    }

The `type` of this task will always be `restore`, and usually,
`log` will be empty and `stopped_at` will have no value, since
the task hasn't had time to run to completion.

**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `operator` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve backup archive information**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such backup archive**:
  The requested backup archive was not found in the database, or
  it was not associated with the given tenant.

- **Unable to schedule a restore task**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### DELETE /v2/tenants/:tenant/archives/:uuid

Remove an archive from a tenant, and purge the archive
data from the backing storage system.


**Request**

    curl -H 'Accept: application/json' \
         -X DELETE https://shield.host/v2/tenants/:tenant/archives/:uuid \


This endpoint takes no query string parameters.

**Response**

    {
      "ok": "Archive deleted successfully"
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `operator` role on the tenant.

**Errors**

The following error messages can be returned:

- **Unable to retrieve backup archive information**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such backup archive**:
  The requested backup archive was not found in the database, or
  it was not associated with the given tenant.

- **Unable to delete backup archive**:
  an internal error occurred and should be investigated by the
  site administrators

- **The backup archive could not be deleted at this time.**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


## SHIELD Global Resources

Some resources are shared between tenants, either implicitly via
copying (like retention policies), or explicitly (like shared
storage system definitions).


### GET /v2/global/stores

Retrieve all globally-defined stores.


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/global/stores


- **?exact=(t|f)**
When filtering stores, perform either exact field / value
matching (`exact=t`), or fuzzy search (`exact=f`, the
default)


- **?unused=(t|f)**
When filtering stores, skip those that are unused (true) or used (false)


- **?name=...**
Only show stores whose name matches the given value.
Subject to the `exact=(t|f)` query string parameter.


- **?plugin=...**
Only show stores who are associated with the given plugin.
Subject to the `exact=(t|f)` query string parameter.


- **?limit=N**
Limit the returned result set to the first _limit_ users
that match the other filtering rules.  A limit of `0` (the
default) denotes an unlimited search.



**Response**

    [
      {
        "uuid"    : "925c83ad-22e6-4cdd-bf63-6dd6d09cd86f",
        "name"    : "Cloud Storage Name",
        "global"  : true,
        "summary" : "A longer description of the storage configuration",
        "agent"   : "127.0.0.1:5444",
        "plugin"  : "fs",
        "config"  : {
          "base_dir" : "/var/data/root",
          "bsdtar"   : "bsdtar"
        }
      }
    ]

The values under `config` will depend entirely on what the
operator specified when they initially configured the storage
system.

**Access Control**

You must be authenticated to access this API endpoint.

**Errors**

The following error messages can be returned:

- **Unable to retrieve storage systems information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.


### POST /v2/global/stores

Create a new shared storage system.  This storage will be visible
to all tenants.


**Request**

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


The values under `config` will depend entirely on which `plugin`
has been selected; no validation will be done by the SHIELD Core,
until the storage system is used in a job.

**Response**

    {
      "name"    : "Storage System Name",
      "summary" : "A longer description for this storage system.",
      "plugin"  : "plugin-name",
      "agent"   : "127.0.0.1:5444",
      "config"  : {
        "plugin-specific": "configuration"
      }
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `engineer` system role.

**Errors**

The following error messages can be returned:

- **Unable to retrieve storage system information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unable to create new storage system**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### GET /v2/global/stores/:uuid

Retrieve a single globally-defined storage system.


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/global/stores/:uuid


This endpoint takes no query string parameters.

**Response**

    {
      "uuid"    : "925c83ad-22e6-4cdd-bf63-6dd6d09cd86f",
      "name"    : "Cloud Storage Name",
      "global"  : true,
      "summary" : "A longer description of the storage configuration",
      "plugin"  : "fs",
      "agent"   : "127.0.0.1:5444",
      "config"  : {
        "base_dir" : "/var/data/root",
        "bsdtar"   : "bsdtar"
      }
    }

The values under `config` will depend entirely on what the
operator specified when they initially configured the storage
system.

**Access Control**

You must be authenticated to access this API endpoint.

**Errors**

The following error messages can be returned:

- **Unable to retrieve storage system information**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such storage system**:
  No storage system with the given UUID exists
  (globally).

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.


### PUT /v2/global/stores/:uuid

Update an existing globally-defined storage system.


**Request**

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


You can specify as many or few of these fields as you want;
omitted fields will be left at their previous values.  If `config`
is supplied, it will overwrite the value currently in the
database.

The values under `config` will depend entirely on which `plugin`
has been selected; no validation will be done by the SHIELD Core,
until the storage system is used in a job.

**Response**

    {
      "name"    : "Updated Store Name",
      "summary" : "A longer description of the storage system",
      "agent"   : "127.0.0.1:5444",
      "plugin"  : "plugin",
      "config"  : {
        "new": "plugin configuration"
      }
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `engineer` system role.

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

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### DELETE /v2/global/stores/:uuid

Remove a globally-defined storage system.


**Request**

    curl -H 'Accept: application/json' \
         -X DELETE https://shield.host/v2/global/stores/:uuid \


This endpoint takes no query string parameters.

**Response**

    {
      "ok": "Storage system deleted successfully"
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `engineer` system role.

**Errors**

The following error messages can be returned:

- **Unable to retrieve storage system information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Unable to delete storage system**:
  an internal error occurred and should be investigated by the
  site administrators

- **The storage system cannot be deleted at this time**:
  This storage system is referenced by one or more
  extant job configuration; deleting it would lead to
  an incomplete (and unusable) setup.

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### GET /v2/global/policies

Retrieve all defined retention policy templates.


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/global/policies


- **?exact=(t|f)**
When filtering policies, perform either exact field / value
matching (`exact=t`), or fuzzy search (`exact=f`, the
default)


- **?unused=(t|f)**
When filtering policies, skip those that are unused (true) or used (false)


- **?name=...**
Only show policies whose name matches the given value.
Subject to the `exact=(t|f)` query string parameter.


- **?limit=N**
Limit the returned result set to the first _limit_ users
that match the other filtering rules.  A limit of `0` (the
default) denotes an unlimited search.



**Response**

    [
      {
        "uuid"    : "f4dedf80-cdb2-4c81-9a58-b3a8282e3202",
        "name"    : "Long-Term Storage",
        "summary" : "A long-term solution, for infrequent backups only.",
        "expires" : 7776000
      }
    ]

The `expires` key is specified in seconds, but must always be a
multiple of 86400 (1 day).

**Access Control**

You must be authenticated to access this API endpoint.

**Errors**

The following error messages can be returned:

- **Unable to retrieve retention policy templates information**:
  an internal error occurred and should be investigated by the
  site administrators

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.


### POST /v2/global/policies

Create a new retention policy template.


**Request**

    curl -H 'Accept: application/json' \
         -H 'Content-Type: application/json' \
         -X POST https://shield.host/v2/global/policies \
         --data-binary '
    {
      "name"    : "Retention Policy Name",
      "summary" : "A longer description of the policy",
      "expires" : 86400
    }'


The `expires` value must be specified in seconds, and must be at
least 86,400 (1 day) and be a multiple of 86,400.

**Response**

    {
      "uuid"    : "4882b332-6182-4123-984f-f9e5dd8dae20",
      "name"    : "Retention Policy Name",
      "summary" : "A longer description of the policy",
      "expires" : 86400
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `engineer` system role.

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

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### GET /v2/global/policies/:uuid

Retrieve a single retention policy template.


**Request**

    curl -H 'Accept: application/json' \
            https://shield.host/v2/global/policies/:uuid


This endpoint takes no query string parameters.

**Response**

    {
      "uuid"    : "4882b332-6182-4123-984f-f9e5dd8dae20",
      "name"    : "Retention Policy Name",
      "summary" : "A longer description of the policy",
      "expires" : 86400
    }
**Access Control**

You must be authenticated to access this API endpoint.

**Errors**

The following error messages can be returned:

- **Unable to retrieve retention policy template information**:
  an internal error occurred and should be investigated by the
  site administrators

- **No such retention policy template**:
  No retention policy template with the given UUID
  exists globally.

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.


### PATCH /v2/global/policies/:uuid

Update a single retention policy template.


**Request**

    curl -H 'Accept: application/json' \
         -H 'Content-Type: application/json' \
         -X PATCH https://shield.host/v2/global/policies/:uuid \
         --data-binary '
    {
      "name"    : "Updated Retention Policy Name",
      "summary" : "A longer description of the retention policy",
      "expires" : 86400
    }'


You can specify as many or few of these fields as you want;
omitted fields will be left at their previous values.

**NOTE:** Updating a retention policy template will not affect any
tenants created prior to the update; updates will apply to new,
future tenants.

**Response**

    {
      "uuid"    : "4882b332-6182-4123-984f-f9e5dd8dae20",
      "name"    : "Retention Policy Name",
      "summary" : "A longer description of the policy",
      "expires" : 86400
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `engineer` system role.

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

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


### DELETE /v2/global/policies/:uuid

Remove a retention policy template.

**NOTE:** Removing a retention policy template will not affect any
tenants created prior to the removal; the template will not be
copied into any future tenants.


**Request**

    curl -H 'Accept: application/json' \
         -X DELETE https://shield.host/v2/global/policies/:uuid \


This endpoint takes no query string parameters.

**Response**

    {
      "ok": "Retention policy template deleted successfully"
    }
**Access Control**

You must be authenticated to access this API endpoint.

You must also have the `engineer` system role.

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

- **Authorization required**:
  The request was made without an authenticated session or auth token.
  See **Authentication** for more details.  The request may be retried
  after authentication.

- **Access denied**:
  The requester lacks sufficient tenant or system role assignment.
  Refer to the **Access Control** subsection, above.
  The request should not be retried.


