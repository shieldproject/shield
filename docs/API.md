# SHIELD API




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

TBD

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

**Errors**

The following error messages can be returned:

- **failed to check SHIELD health** - an internal error occurred
  and shoud be investigated by the site administrators.




## SHIELD Authentication

TBD

### POST /v2/auth/login

Authenticate against the SHIELD API as a local user, and retrieve
a session ID that can be used for future, authenticated,
interactions.

**Request**

```sh
curl -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -X POST https://shield.host/v2/auth/login -d '
{
  "username": "your-username",
  "password": "your-password"
}'
```

Note: `password` is sent in cleartext, so SHIELD should aways be
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

**Errors**

The following error messages can be returned:

- **no username given** - The required field `username` was not
  supplied.  Note that this is errant behavior, inconsistent with
  the rest of the SHIELD API.  It should be FIXME'd before v8.
- **no password given** - The required field `password` was not
  supplied.  Note that this is errant behavior, inconsistent with
  the rest of the SHIELD API.  It should be FIXME'd before v8.
- **Unable to authenticate** - An internal error occurred and
  should be investigated by the site administrators.
- **Incorrect username or password** - The supplied credentials
  were incorrect; either the user doesn't exist, or the password
  was wrong.
- **Unable to create session** - An internal error occurred and
  should be investigated by the site administrators.
- FIXME - one more error, but waiting on Tom to submit a PR.




### GET /v2/auth/logout

TBD

**Request**

TBD

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### GET /v2/auth/id

TBD

**Request**

TBD

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:




## SHIELD Core

TBD

### POST /v2/init

Initializes a new SHIELD Core, to set up the encryption facilities
for storing backup archive encryption keys safely and securely.
Your SHIELD Core can only be initialized once.

**Request**

```sh
curl -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -X POST https://shield.host/v2/init -d '
{
  "master" : "your secret master password"
}'
```

Where:

- **master** is the plaintext master password to use for
  encrypting the credentials to the SHIELD Core storage vault.

**Response**

If all went well, and the SHIELD Core was properly initialized,
you will receive a 200 OK, and the following response:

```json
{
  "ok" : "Successfully initialized the SHIELD Core"
}
```

**Errors**

The following error messages can be returned:

- **Unable to initialize the SHIELD Core** - An internal error
  occurred and should be investigated by the site administrators.
- **This SHIELD Core has already been initialized** - You are
  attempting to re-initialize a SHIELD Core, which is not allowed.

### POST /v2/unlock

Unlocks a SHIELD Core by providing the master password.  This
allows SHIELD to decrypt the keys to access its storage vault and
generate / retrieve backup archvie encryption keys safely and
securely.

**Request**

```sh
curl -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -X POST https://shield.host/v2/unlock -d '
{
  "master" : "your secret master password"
}'
```

- **master** is the plaintext master password that was
  created when you initialized this SHIELD Core (or whatever you
  last rekeyed it to be).

**Response**

On success, you will receive a 200 OK, with the following
response:

```json
{
  "ok" : "Successfully unlocked the SHIELD Core"
}
```

**Errors**

The following error messages can be returned:

- **Unable to unlock the SHIELD Core** - An internal error
  occurred and should be investigated by the site administrators.
- **This SHIELD Core has not yet been initialized** - You may
  re-attempt this request after initializing the SHIELD Core.

### POST /v2/rekey

Changes the master password used for encrypting the credentials
for the SHIELD Core storage vault (where backup archive encryption
keys are held).

**Request**

```sh
curl -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -X POST https://shield.host/v2/unlock -d '
{
  "current" : "your CURRENT master password",
  "new"     : "what you want to change it to"
}'
```

**Response**

If all goes well, you will receive a 200 OK, and the following
response:

```json
{
  "ok" : "Successfully rekeyed the SHIELD core"
}
```

**Errors**

The following error messages can be returned:

- **Unable to rekey the SHIELD Core** - An internal error occurred
  and should be investigated by the site administrators.




## SHIELD Agents

TBD

### GET /v2/agents

Retrieves information about all registered SHIELD Agents.

**Request**

```sh
curl -H 'Accept: application/json' \
     -X POST https://shield.host/v2/agents
```

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


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
curl -H 'Content-Type: application/json' \
     -X POST https://shield.host/v2/agents -d '
{
  "name" : "some-identifier",
  "port" : 5444
}'
```

Where:

- **name** is the name of the agent to display in the backend, and
  in log messages.  Usually, an FQDN or other unique host
  identifier is preferable here.
- **port** is the port number that the SHIELD agent is bound to.
  The remote peer IP will be determined from the HTTP request's
  peer address.

**Response**

On success, you will receive a 200 OK, and the following response:

```json
{
  "ok" : "Pre-registered agent <name> at <host>:<port>"
}
```

**Errors**

The following error messages can be returned:

- **No \`name' provided with pre-registration request** - Your
  request was missing the required `name` argument.  Re-attempt
  with the `name` argument.
- **No \`port' provided with pre-registration request** - Your
  request was missing the required `port` argument.  Re-attempt
  with the `port` argument.
- **Unable to pre-register agent \<name\> at \<host\>:\<port\>** -
  An internal error occurred and shoud be investigated by the site
  administrators.
- **Unable to determine remote peer address from '\<peer\>'** -
  SHIELD was unable to parse the HTTP connection's peer address as
  a valid IP address.  This should be investigated by the site
  administrators, your local network administrator, and possibly
  the SHIELD development team.



### GET /v2/agents/:uuid

TBD

**Request**

```sh
curl -H 'Accept: application/json' \
     -X POST https://shield.host/v2/agents/$uuid
```

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:




## SHIELD Tenants

TBD


### GET /v2/tenants

```sh
curl -H 'Accept: application/json' \
     -X POST https://shield.host/v2/tenants
```

**Request**

TBD

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### POST /v2/tenants

TBD

**Request**

TBD

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### GET /v2/tenants/:uuid

TBD

**Request**

```sh
curl -H 'Accept: application/json' \
     -X POST https://shield.host/v2/tenants/$uuid
```

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### PUT /v2/tenants/:uuid

FIXME: is this just a PATCH?

TBD

**Request**

TBD

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### PATCH /v2/tenants/:uuid

TBD

**Request**

TBD

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### POST /v2/tenants/:uuid/invite

TBD

**Request**

TBD

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### POST /v2/tenants/:uuid/banish

TBD

**Request**

TBD

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### DELETE /v2/tenants/:uuid

TBD

**Request**

```sh
curl -H 'Accept: application/json' \
     -X DELETE https://shield.host/v2/tenants/$uuid
```

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:




## SHIELD Targets

TBD


### GET /v2/tenants/:tenant/targets

TBD

**Request**

```sh
curl -H 'Accept: application/json' \
     -X POST https://shield.host/v2/tenants/$tenant/targets
```

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### POST /v2/tenants/:tenant/targets

TBD

**Request**

TBD

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### GET /v2/tenants/:tenant/targets/:uuid

TBD

**Request**

```sh
curl -H 'Accept: application/json' \
     -X POST https://shield.host/v2/tenants/$tenant/targets/$uuid
```

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### PUT /v2/tenants/:tenant/targets/:uuid

TBD

**Request**

TBD

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### DELETE /v2/tenants/:tenant/targets/:uuid

TBD

**Request**

```sh
curl -H 'Accept: application/json' \
     -X DELETE https://shield.host/v2/tenants/$tenant/targets/$uuid
```

**Response**

```json
{
  "ok": "Target delete successfully"
}
```

**Errors**

TBD

The following error messages can be returned:




## SHIELD Stores

TBD


### GET /v2/tenants/:tenant/stores

TBD

**Request**

```sh
curl -H 'Accept: application/json' \
     -X POST https://shield.host/v2/tenants/$tenant/stores
```

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### POST /v2/tenants/:tenant/stores

TBD

**Request**

TBD

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### GET /v2/tenants/:tenant/stores/:uuid

TBD

**Request**

```sh
curl -H 'Accept: application/json' \
     -X POST https://shield.host/v2/tenants/$tenant/stores/$uuid
```

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### PUT /v2/tenants/:tenant/stores/:uuid

TBD

**Request**

TBD

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### DELETE /v2/tenants/:tenant/stores/:uuid

TBD

**Request**

```sh
curl -H 'Accept: application/json' \
     -X DELETE https://shield.host/v2/tenants/$tenant/stores/$uuid
```

**Response**

```json
{
  "ok": "Store delete successfully"
}
```

**Errors**

TBD

The following error messages can be returned:




## SHIELD Retention Policies

TBD


### GET /v2/tenants/:tenant/policies

TBD

**Request**

```sh
curl -H 'Accept: application/json' \
     -X POST https://shield.host/v2/tenants/$tenant/policies
```

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### POST /v2/tenants/:tenant/policies

TBD

**Request**

TBD

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### GET /v2/tenants/:tenant/policies/:uuid

TBD

**Request**

```sh
curl -H 'Accept: application/json' \
     -X POST https://shield.host/v2/tenants/$tenant/policies/$uuid
```

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### PUT /v2/tenants/:tenant/policies/:uuid

TBD

**Request**

TBD

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### DELETE /v2/tenants/:tenant/policies/:uuid

TBD

**Request**

```sh
curl -H 'Accept: application/json' \
     -X DELETE https://shield.host/v2/tenants/$tenant/policies/$uuid
```

**Response**

```json
{
  "ok": "Retention policy delete successfully"
}
```

**Errors**

TBD

The following error messages can be returned:




## SHIELD Jobs

TBD


### GET /v2/tenants/:tenant/jobs

TBD

**Request**

```sh
curl -H 'Accept: application/json' \
     -X POST https://shield.host/v2/tenants/$tenant/jobs
```

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### POST /v2/tenants/:tenant/jobs

TBD

**Request**

TBD

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### GET /v2/tenants/:tenant/jobs/:uuid

TBD

**Request**

```sh
curl -H 'Accept: application/json' \
     -X POST https://shield.host/v2/tenants/$tenant/jobs/$uuid
```

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### PUT /v2/tenants/:tenant/jobs/:uuid

TBD

**Request**

TBD

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### DELETE /v2/tenants/:tenant/jobs/:uuid

TBD

**Request**

```sh
curl -H 'Accept: application/json' \
     -X DELETE https://shield.host/v2/tenants/$tenant/jobs/$uuid
```

**Response**

```json
{
  "ok": "Job delete successfully"
}
```

**Errors**

TBD

The following error messages can be returned:




## SHIELD Tasks

TBD


### GET /v2/tenants/:tenant/tasks

TBD

**Request**

```sh
curl -H 'Accept: application/json' \
     -X POST https://shield.host/v2/tenants/$tenant/tasks
```

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### POST /v2/tenants/:tenant/tasks

TBD

**Request**

TBD

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### GET /v2/tenants/:tenant/tasks/:uuid

TBD

**Request**

```sh
curl -H 'Accept: application/json' \
     -X POST https://shield.host/v2/tenants/$tenant/tasks/$uuid
```

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### PUT /v2/tenants/:tenant/tasks/:uuid

TBD

**Request**

TBD

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### DELETE /v2/tenants/:tenant/tasks/:uuid

TBD

**Request**

```sh
curl -H 'Accept: application/json' \
     -X DELETE https://shield.host/v2/tenants/$tenant/tasks/$uuid
```

**Response**

```json
{
  "ok": "Task canceled successfully"
}
```

**Errors**

TBD

The following error messages can be returned:




## SHIELD Backup Archives

TBD


### GET /v2/tenants/:tenant/archives

TBD

**Request**

```sh
curl -H 'Accept: application/json' \
     -X POST https://shield.host/v2/tenants/$tenant/archives
```

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### POST /v2/tenants/:tenant/archives

TBD

**Request**

TBD

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### GET /v2/tenants/:tenant/archives/:uuid

TBD

**Request**

```sh
curl -H 'Accept: application/json' \
     -X POST https://shield.host/v2/tenants/$tenant/archives/$uuid
```

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### PUT /v2/tenants/:tenant/archives/:uuid

TBD

**Request**

TBD

**Response**

TBD

**Errors**

TBD

The following error messages can be returned:


### DELETE /v2/tenants/:tenant/archives/:uuid

TBD

**Request**

```sh
curl -H 'Accept: application/json' \
     -X DELETE https://shield.host/v2/tenants/$tenant/archives/$uuid
```

**Response**

```json
{
  "ok": "Archive delete successfully"
}
```

**Errors**

TBD

The following error messages can be returned:




