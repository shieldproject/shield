Agent Fabrics
=============

Prior to version 8.1, SHIELD only supported one method of agent
orchestration: an active SSH connection was made from the SHIELD
core every time a task needed executing.

This works great in single-segment networking, when address
translation is not in play.  However, as soon as the agent host
and the SHIELD Core are separated by one or more NAT devices, it
breaks down.

Starting with 8.1, we have abstracted this communication method
out from the core code, to allow alternate systems to be tried out
and implemented, based on site and operator need.  We call these
_fabrics_.


Each _fabric_ implements the `fabric.Fabric` interface, as defined
in `$src/core/fabric/fabric.go`:

    type Fabric interface {
        /* back up a target to a store, encrypt it,
           and optionally compress it. */
        Backup(*db.Task, vault.Parameters) scheduler.Chore

        /* restore an encrypted archive to a target. */
        Restore(*db.Task, vault.Parameters) scheduler.Chore

        /* check the status of the agent. */
        Status(*db.Task) scheduler.Chore

        /* purge an from cloud storage archive. */
        Purge(*db.Task) scheduler.Chore

        /* test the viability of a storage system. */
        TestStore(*db.Task) scheduler.Chore
    }

Each of these methods corresponds to a task type.



Configuring Fabrics
-------------------

With the exception of trivial fabrics (which we'll see in a
moment), all fabrics need some sort of configuration.  To
facilitate this, the SHIELD Core now has a top-level `fabrics:`
configuration directive that houses operator-provided details
about how the fabric in question should behave.

Here is an example configuration snippet, which provides the SSH
private key necessary for the Legacy (Active SSH) fabric to work:

    fabrics:
      - name: legacy
        ssh-key: |
          -----BEGIN RSA PRIVATE KEY-----
          MIIEpQIBAAKCAQEAw1G6CPhJ/+/6WdYGab80FeBU/ERxaYUY7GT3WHnsth1Pw77O
          ... etc. ...
          6JKLdAirjIQB7QVu8DtFFN6gnqrvD+roej55exRlKN8uAGo0VrCo3LQ=
          -----END RSA PRIVATE KEY-----

**Note**: each fabric has bespoke code in the SHIELD core module
to handle this configuration.  If a new fabric is added, the core
needs to be updated accordingly.


Fabric Selection
----------------

TBD



The Dummy Fabric
----------------

For _TESTING PURPOSES ONLY_, the Dummy Fabric implements mock
handlers for all task types.  The fabric as a whole can be
configured with a _delay_ parameter, which it uses to sleep during
task execution.  This can be quite handy for testing out the
SHIELD scheduler, since it slows down the normally fast-paced
scheduling and execution activities, so you can observe the
scheduler and its priority queue.

Configuration is as follows:

    fabrics:
      - name:  dummy
        delay: 15 # seconds

The Dummy Fabric is defined in `$src/core/fabric/dummy.go`.



The Error Fabric
----------------

The Error fabric exists mostly as an internal implementation
detail.  Operators should never configure this fabric in their
SHIELD cores.  Every method results in a failed task.

The Error Fabric is defined in `$src/core/fabric/error.go`.



The Legacy (Active SSH) Fabric
------------------------------

The Legacy Fabric provides the active SSH connection method of
agent orchestration that was built into SHIELD versions prior to
8.1.  Every task is executed across an SSH `exec` channel.

To configure the Legacy fabric, the SHIELD operator has to provide
the private SSH key to use for initiating connections to the
listening agents:

    fabrics:
      - name: legacy
        ssh-key: |
          ... an RSA private key, in PEM format ...

The Legacy Fabric is defined in `$src/core/fabric/legacy.go`.
