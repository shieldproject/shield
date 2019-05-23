SHIELD Architecture
===================

![SHIELD Architectural Overview](overview.png)

Feel free to skim this section on a first reading.  It is definitely
worth a thorough review, however, especially given how pervasive
these ideas are in SHIELD.

## Core

<img src="core.png" style="float: right; margin: 0 0 32px 32px;">

The **SHIELD Core** is the heart, mind, and soul of a SHIELD installation.
It is responsible for scheduling, metadata, configuration, monitoring, and
much more.

The core is composed of several discrete components that all cooperate to
deliver on the promise of data protection.  They include:

  1. The **SHIELD Database**
  2. The **SHIELD Vault**
  3. The **SHIELD Scheduler**
  4. **SHIELD Workers**
  5. The **SHIELD API**

These are described below, in more detail.

## Agents

<img src="agent.png" style="float: right; margin: 0 0 32px 32px;">

Without **SHIELD Agents**, nothing would get done.  These distributed bits
of software are reponsible for executing all manner of tasks, from backups,
to restores, to archive purgation and validation testing.

Each agent uses locally installed _plugins_ to interact with _targets_ and
_stores_ to run backups, perform restores, prune archives, etc.

## Plugins

**Plugins** enable SHIELD to communicate with several different target data
systems, and cloud storage solutions.  These standalone
executables operate according to a known protocol,

// FIXME document the SHIELD plugin protocol

and allow operators and system integrators to support data systems and
storage backends we haven't even dreamt up yet.

For a list of the currently implemented plugins that ship with SHIELD
itself, see our [plugin documentation]($docs/ops/plugins).

## Targets

A **target** is a data system, and they vary wildly.  A PostgreSQL database
can be a target.  So can a Consul key-value store, or a Redis installation.
If there's data (or live configuration that you don't want to lose), you can
bet SHIELD thinks of it as a target.

SHIELD Agents interact with targets via _target plugins_.

## Stores

A **store** is a storage system, usually off-site, redundant, remote, or all
three.  SHIELD stores archives in these systems, and then retrieves those
archives for restoration tasks, later.  Prominent, well-known stores include
Amazon S3, WebDAV, Backblaze, etc.

SHIELD Agents interact with stores via _store plugins_.


## Scheduler

<img src="scheduler.png" style="float: right; margin: 0 0 32px 32px;">

At the heart of the SHIELD Core is the scheduler.  It is responsible for
regularly checking all defined _jobs_ to see if they need to be executed (as
_tasks_).  It also handles a series of other tasks, including storage
tests, archive pruning, etc.

## Jobs

A **job** ties together a _target_ data system, a cloud _store_, and
supplies scheduling and archive retention configuration.  When scheduled, a
job becomes a _task_.

## Tasks

The smallest schedulable unit in SHIELD, a **task** represents some specific
operation, with configuration values, that will be executed at most once.  A
_job_ turns into a _task_ when it is scheduled by the SHIELD Core.

SHIELD also uses tasks for other purposes; pruning archives that have
outlived their retention policy is handled via _prune_ tasks.  Cloud
_stores_ are validated regularly via _test-store_ tasks.

## Archives

A **backup archive** is the output of a successful backup task, and contains
the encrypted and optionally compressed data that was extracted from the
_target_ data system.  Archives are kept in cloud _stores_ until they
outlive their expiration, as set by the _job_ that effected the _task_.

## Workers

<img src="worker.png" style="float: right; margin: 0 0 32px 32px;">

Internally, the SHIELD Core runs a set of worker threads that manage
all communication with SHIELD Agents.  These workers handle the execution of
tasks on specific agents, stream logs back into the core, update task status
as execution proceeds, and more.

## Vault

<img src="vault.png" style="float: right; margin: 0 0 32px 32px;">

The **SHIELD Vault** is a secure credentials storage solution. SHIELD
uses the vault to store sensitive information like encryption
parameters.

## Web Interface

<img src="webui.png" style="float: right; margin: 0 0 32px 32px;">

The **Web Interface** (or just _web UI_) is a graphical, web-based
management interface for interacting with SHIELD.  It provides modest
monitoring capabilities, and allows operators to configure their SHIELD
installations.

## CLI

<img src="cli.png" style="float: right; margin: 0 0 32px 32px;">

The **SHIELD CLI** is a command-line interface to SHIELD.  It allows
operators who do not wish to use a graphical interface to manager their
SHIELD installations.

The CLI can also be put to use in a variety of automation context, as it is
built to be scripted (as long as you can grok JSON output).

## API

<img src="api.png" style="float: right; margin: 0 0 32px 32px;">

SHIELD features a rich and robust REST API, using JSON as a baseline payload
format for both requests and responses.  All approved external management
utilities, including the Web UI and the CLI, exclusively use this API to do
their jobs, ensuring that the API is featureful and complete.

## Database

<img src="database.png" style="float: right; margin: 0 0 32px 32px;">

The **SHIELD Database** is where non-sensitive, persistent data is stored.
This includes everything from target configuration, to job schedules, to
task histories and logs.

SHIELD uses the excellent SQLite3 embedded database system.
