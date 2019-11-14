# New Features

- Helm Support!  This version of SHIELD ships with OCI Docker
  images that can be used in the new (Beta!) helm chart for
  SHIELD.  See https://github.com/shieldproject/helm for more
  details, and to give it a spin yourself.

- The SHIELD Core can now be configured almost entirely through
  environment variables, for ease of configuration in Docker,
  Compose, and even Kubernetes.

- We have a new Prometheus-compatible metrics exporter, accessible
  at `/metrics`, and governed by a separate set of HTTP Basic Auth
  credentials.

# Improvements

- Agent SSH is now constrained to a more secure set of message
  authentication codes (MACs).  Specifically, we got rid of one
  embarassing 96-bit MAC algorithm.  Ooof!

- Several quality-of-life improvements were made to the web UI
  and message bus / websocket implementations.  In general, the
  web interface is easier to use and more robust now.

- Old task logs and purged archives will now be removed from the
  database after a minimum retention period has passed.  If you've
  been with us since the 0.x days, this update is for you, and
  we're sorry it's taken us so long to do this type of cleanup.

- The SHIELD IP Address (which gets less and less relevant every
  day) is no longer reported via the API / web UI.

# Bug Fixes

- Uncompressed backups can now properly be restored.

- The `healthy` and `paused` fields of the Jobs table now no
  longer allows NULL values, landing us squarely back in the
  territory of booleanitude -- things are either true or false;
  there is no maybe.

- Negative daily storage increases now properly convert to kilo-,
  mega-, and giga- units, to help humans understand magnitude.

- The `api.session.timeout` value is now interpreted properly as
  seconds, not hours.  This effectively means that sessions now
  expire when they ought to, not several orders of magnitude
  later.
