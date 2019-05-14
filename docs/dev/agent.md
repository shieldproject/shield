SHIELD Agent Protocol
=====================

The Agents of SHIELD are the workhorse of the data protection
solution.  They perform backup and restore tasks.

All communication with the Agents is initiated by the Core.
Agents **never** initiate a connection to the Core themselves.

All communication is handled over an SSH connection.  This
provides endpoint identity validation (keys must match) and
confidentiality (so that credentials to target / storage systems
are not compromised).  The SSH protocol is documented fully in
[RFC 4251 (Architecture)][rfc4251], [RFC 4252 (AUTH
Protocol)][rfc4252], [RFC 4253 (Transport Layer)][rfc4253], and
[RFC 4254 (Connection Protocol)][rfc4254].

Communication between the Core and the Agent is intermittent; the
Core does **not** maintain a persistent SSH connection to any
Agent.  Instead, as tasks are run on by the Core Scheduler,
connections are made to the involved Agents as needed.

SHIELD uses the [Connection Protocol (RFC 4254)][rfc4254] to issue
commands from the SHIELD Core to its participating Agents.  Each
connection from the Core to an Agent initiates a new SSH Session.
Once that Session is fully set up, an `SSH_MSG_CHANNEL_REQUEST` is
sent, and a new Channel, of type "exec" is established.  The
_command_ string sent with this channel is a JSON-encoded
Agent-Request.

(For more information on SSH channels, refer to RFC 4254).

The format of an Agent-Request is as follows:

    {
      "operation"       : "OP-STRING",

      "target_plugin"   : "PLUGIN-NAME",
      "target_endpoint" : "JSON-encoded STRING",

      "store_plugin"    : "PLUGIN-NAME",
      "store_endpoint"  : "JSON-encoded STRING",

      "restore_key"     : "OPAQUE-IDENTIFIER",
    }

As of SHIELD v0.10.8, the only recognized operations are:

  - backup
  - restore
  - purge

Future Direction
----------------

For the 7.x version of SHIELD, the [ROADMAP][map] lays out several
ambitious usability goals that require changes to the Agent
Protocol.  In order to give operators and site administrators the
smoothest possible roll-out, we need to be highly aware of
backwards-compatibility at every turn.

The following new functionality needs to be added:

- **Metadata** - Agents must respond to metadata queries from the
  SHIELD core and provide information about themselves, including
  identifiers (name / address), version details, available plugins
  (and _their_ versions), metadata for how to configure each
  plugin, etc.

- **Storage Tests** - Agents must respond to "test" requests by
  running plugin-specific health checks against the given endpoint
  configurations.  This helps operators answer questions like "can
  I retrieve archives from S3?"

- **Plugin Validation** - Agents must respond to "validate"
  requests by running plugin validation for the given plugin,
  against the given configuration.

To this end, Agents will need to be able to handle the following
new request types:

### "status"

The SHIELD Core requests Agent status by issuing the following
Agent-Request:

    {
      "operation" : "status"
    }

Older SHIELD agents will respond to this with an error.  SHIELD
Core will have to mark the agent as v0.0.0 (or whatever the most
recent version before agents learned how to respond properly), and
treat the health as "degraded"

Newer SHIELD agents will respond with the following JSON:

    {
      "name"    : "<name-set-by-admin>",
      "version" : "<X.Y.Z>",
      "health"  : "ok",
      "plugins" : {
        "s3" : {
          "name"    : "S3 Backup + Storage Plugin",
          "author"  : "Stark \u0026 Wayne",
          "version" : "0.0.1",
          "features": {
              "target" : "no",
              "store"  : "yes"
          },
          "config" : {
            "store" : [
              { FIELD-DEFINITION },
              ...
            ]
          }
        },
        ...
      }
    }

The `name` and `version` field are just strings.  The Web UI will
interpret `version` and search for appropriate misconfigurations
and other pitfalls between Core version and Agent version.

The `health` field indicates an Agent-defined assertion of the
overall health of the Agent.  The specifics of what goes into
making this decision are left to the Agent implementation.  The
possible values are:

- "ok" - Everything is good; no problems detected.
- "degraded" - Non-fatal issues were detected.  For example,
  some plugin scripts may not be executable.
- "failing" - Fatal issues were detected.  No plugins, invalid
  plugins detected, other environmental issues, etc.

(Note that a SHIELD core may upgrade the critical-ness of a health
alert.  For example, a non-responsive agent may be marked as
"failing")

The `plugins` field contains a map of plugin metadata, keyed by
executable name.  This allows the Core to build up a list of
capabilities for each agent, and validate configuration before it
is attempted on-Agent.

The values of the `plugins` map are the JSON output by executing
the `info` command.  Legacy plugin binaries will not include the
`config` top-level key; that is a new addition.

### "test"

The SHIELD Core requests a plugin configuration test by issuing
the following Agent-Request:

    {
      "operation" : "test"
      "plugin"    : "<PLUGIN-NAME>"
      "endpoint"  : {
         ...
      }
    }

Older SHIELD agents will respond to this with an error.  SHIELD
Core must interpret this as a non-blocking failure of plugin
testing.  For purposes of gauging Cloud Storage Health, the
storage plugin should be considered failed.  Other circumstances
may call for more or less caution.

Newer SHIELD agents will respond with the following JSON:

    {
      "status"  : "ok",
      "message" : "Targeting S3 at s3.example.com"
    }

or, in the event of failure:

    {
      "status"  : "failed",
      "message" : "Unable to contact s3.example.com - connection refused"
    }

The only two valid values for `status` are `"ok"` and `"failed"`.

The `message` can be displayed to the end user if the UI deems it
useful and appropriate.  Plugin authors are discouraged from
embedding sensitive information in the status message.

### "validate"

The SHIELD Core requests validation of a given plugin + endpoint
by issuing the following Agent-Request:

    {
      "operation" : "validate"
      "plugin"    : "<PLUGIN-NAME>"
      "endpoint"  : {
         ...
      }

A _validation_ is run by the plugin against the endpoint
configuration (passed here as a first-class map, instead of a
doubly JSON-encoded representation).

Older SHIELD agents will respond to this with an error.  SHIELD
Core must interpret that as a nonblocking failure.  The
configuration _may_ be valid, but the Agent is unable to perform
the validation, so it is unknown.

Newer SHIELD agents will respond with the output of the plugin
`validate` execution run, and a channel exit code of 0 for success
and non-zero for failure.  (This is how existing tasks are
reported)



[rfc4251]: https://tools.ietf.org/rfc/rfc4251.txt
[rfc4252]: https://tools.ietf.org/rfc/rfc4252.txt
[rfc4253]: https://tools.ietf.org/rfc/rfc4253.txt
[rfc4254]: https://tools.ietf.org/rfc/rfc4254.txt

[map]: https://github.com/starkandwayne/shield/blob/master/ROADMAP.md
