SHIELD Roadmap
==============

This document contains the current SHIELD Roadmap, a set of things
we hope to accomplish, features we hope to implement, and code
management / maintenance efforts we hope to undertake in 2017/18.

Roadmap items are listed in order of importance, roughly.  If you
would like to see ammendments to this Roadmap, please discuss them
in the [#shield Slack channel][1]

Encryption of Archives
----------------------

For security and peace of mind, operators would like to have their
backup archives encrypted, prior to storage in the backend storage
system.  Key features:

**Upgradable Encryption** - As time moves on, operators will need
to be able to upgrade to stronger cryptographic primitives as
current best-practice algorithms are broken and fall out of favor.
These changes must only affect future archives, such that extant
archives are still able to be decrypted using the same algorithm
that they were encrypted with.

**Managed Key Rotation** - Ideally, each backup task would get a
new encryption key to mitigate the damage of a leaked key.  The
"smoking hole" scenario presents some interesting challenges with
regards to restoring the SHIELD database (and by extension, all
encryption keys) from nothing.

User-Driven Backup and Restore
------------------------------

Currently, SHIELD has a flat access model, an all-or-nothing
proposition for review and management of jobs, tasks, archives,
schedules, etc.

We would like to move to a more robust access model that permits
individual customers, be they operators or develoeprs, to view and
control a limited subset of the data set, and effect backup and
restore operations on owned jobs and target systems.

Ideally, non-administrator users will be able to:

  - Create stores, targets, schedules, and retention policies
  - Schedule and run backup jobs
  - Restore their data from their backup archives
  - Review logs for scheduled and ad hoc tasks
  - Utilize global stores, targets, schedules, and policies

Both the web interface and the `shield` command-line utility will
need to support this isolation of resources.

In addition to user-level access, SHIELD needs to support _teams_,
allowing multiple users to share management of a set of resources.

Documentation
-------------

SHIELD documentation is abysmal.  We need to fix this.

To wit, the following documentation needs to be developed:

**Intro / Getting Started** - A short introductory document that
explains what SHIELD is, why it exists, what benefits it provides,
and how it operates.

**Installation / Setup** - A walkthrough of installing SHIELD in a
variety of scenarios, including manual deployment, BOSH, and
Docker Compose.  Should also introduce the SHIELD CLI and the web
user interface, to give the reader enough of a starting point for
future discovery and exploration.

**Backing up and Restoring Data** - A step-by-step guide to
configuring backup jobs in a running SHIELD.  Assumes that the
reader has completed the Installation / Setup guide.  Should focus
on a single target / store plugin pair (i.e. PostgreSQL + S3) and
cover topics like scheduling, retention policies, target and store
configuration, ad hoc jobs, archive purgation, and manual
restores.  Preferably, the backup and restore steps are
intertwined with data creation / destruction steps, so that the
reader can actually see the restore working.

**Supported Data Systems** - A more thorough treatment of the
available backup/restore plugins, what they can and cannot do, and
how they operate.  Readers should go away with a firm grasp on the
capabilities of target- and store-plugins, and be well-equipped to
solve problems in the real world.

**Disaster Recovery of SHIELD Itself** - A short process document
that covers the "smoking hole" scenario of disaster recovery and
restoration for a _SHIELD itself_.

**Contributing to SHIELD** - A developer-oriented document
describing how to develop SHIELD, how to deploy it locally for
tinkering purposes, and how to get contributions into the mainline
repository.


Visual Aesthetic
----------------

The SHIELD web UI is barely serviceable.  It requires an overhaul
of the UI/UX and visual elements to make it both handsome and
usable.  UX areas to focus on include navigation, embedded help,
and feedback loops.  UI areas to focus on include logo work,
consistent color usage, and mobile presentation.

Web Presence / Branding
-----------------------

SHIELD is getting a website!

Archive Download
----------------

Often, operators would like to manually inspect and verify a
backup archive before they restore it to the target system.
Insofar as this is supported, it requires specialized knowledge of
both SHIELD internals, and the storage system holding the backup
archives.

Ideally, there should be a means for an authenticated and entitled
user to retrieve the contents of the backup archive via either the
CLI (`shield download ...`) or the web interface.  With the
introduction of encryption (see above), we will need to ensure
that archives can be downloaded in both encyphered and decyphered
forms.

Monitoring
----------

Integrate SHIELD into major monitoring platforms, to ensure that
operators can detect _immediate failures_ and _brewing problems_.
"Immediate failures" include failing jobs, paused jobs that ought
not be paused, and (if possible) target systems that are not
scheduled for backup.  "Brewing problems" are more
metric-oriented, including things like storage used,
time-to-backup, etc.

Agent Discovery and Capabilities
--------------------------------

SHIELD supports remote agent backup / restore execution, whereby a
backup / restore operation can be exeucted on the host holding the
data system.  To utilize this currently requires _a priori_
knowledge of the network presence of agents, their IP addresses
and TCP ports.

Ideally, the CLI and web interface would be able to show the
operator the available agents, by hostname, UUID, or some other
human-friendly identifier.  From there, an operator should be able
to view what local SHIELD plugins are installed, at what versions,
and what their capabilities are.

Form-Driven Plugin Configuration
--------------------------------

Today, configuring a target or store requires both detailed
knowledge of the plugins required and optional configuration
directives, and a working knowledge of JSON formatting
requirements.

We would like to move to a more human-friendly configuration, in
which operators are asked for specific pieces of information when
they configure a target or store.

For example, when configuring the `fs` plugin, the web interface
currently presents a textarea into which the operator must input
well-formed JSON.  It would be preferable to present a form that
asks for the base directory to backup, include and exclude
patterns, etc.  These form elements can be accompanied both by
validation routines (paths must be absolute; ports, numeric) and
embedded help / examples.

Plugin PATH Environments
------------------------

Several plugins have developed a nasty pattern of specifying
explicit paths to binaries used by the plugin.  For example, the
`fs` plugin has a `bsdtar` configuration option that holds the
path to the BSD distribution of `tar`.  This defaults to a value
that is useful for BOSH-deployed SHIELD agents, but has little
chance of working in other environments.

The original impetus for this design decision was to ensure that
the agent was not dependent on wonky `$PATH` configurations.  In
hindishgt, a better approach is to have the agent configure its
`$PATH` explicitly, from configuration, and expose the correct
paths that way.

So let's do that.



[1]: https://cloudfoundry.slack.com/messages/shield
