# SHIELD API




## Error Handling




## Health

### GET /v2/health

Returns health information about the SHIELD Core, connected
storage accounts, and general metrics.

**Request**

```sh
curl -H 'Accept: application/json' https://shield.host/v2/health
```

#### Response

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

#### Errors

The following error messages can be returned:

- **failed to check SHIELD health** - an internal error occurred
  and shoud be investigated by the site administrators.




## SHIELD Core

### POST /v2/init

Initializes a new SHIELD Core, to set up the encryption facilities
for storing backup archive encryption keys safely and securely.
Your SHIELD Core can only be initialized once.

#### Request

```sh
curl -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -X POST https://shield.host/v2/init -d '
{
  "master_password" : "your secret master password"
}'
```

Where:

- **master_password** is the plaintext master password to use for
  encrypting the credentials to the SHIELD Core storage vault.

#### Response

If all went well, and the SHIELD Core was properly initialized,
you will receive a 200 OK, and the following response:

```json
{
  "ok" : "Successfully initialized the SHIELD Core"
}
```

#### Errors

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

#### Request

```sh
curl -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -X POST https://shield.host/v2/unlock -d '
{
  "master_password" : "your secret master password"
}'
```

- **master_password** is the plaintext master password that was
  created when you initialized this SHIELD Core (or whatever you
  last rekeyed it to be).

#### Response

On success, you will receive a 200 OK, with the following
response:

```json
{
  "ok" : "Successfully unlocked the SHIELD Core"
}
```

#### Errors

The following error messages can be returned:

- **Unable to unlock the SHIELD Core** - An internal error
  occurred and should be investigated by the site administrators.
- **This SHIELD Core has not yet been initialized** - You may
  re-attempt this request after initializing the SHIELD Core.

### POST /v2/rekey

Changes the master password used for encrypting the credentials
for the SHIELD Core storage vault (where backup archive encryption
keys are held).

#### Request

```sh
curl -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -X POST https://shield.host/v2/unlock -d '
{
  "current_master_password" : "your CURRENT master password",
  "new_master_password"     : "what you want to change it to"
}'
```

#### Response

If all goes well, you will receive a 200 OK, and the following
response:

```json
{
  "ok" : "Successfully rekeyed the SHIELD core"
}
```

#### Errors

The following error messages can be returned:

- **Unable to rekey the SHIELD Core** - An internal error occurred
  and should be investigated by the site administrators.




## SHIELD Agents

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

#### Request

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

#### Response

On success, you will receive a 200 OK, and the following response:

```json
{
  "ok" : "Pre-registered agent <name> at <host>:<port>"
}
```

#### Errors

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
