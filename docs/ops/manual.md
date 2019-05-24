SHIELD Operator's Manual
========================

Hello, and welcome to the **SHIELD Operator's Manual**, an in-depth look at
all things SHIELD.  This guide aims to be an exhaustive guide to the
installation and operation of the SHIELD Data Protection solution.

If you are looking for a more easy-going start-up guide, you may want to
check out the [Getting Started]($docs/ops/getting-started) guide.

If you are interested in contributing to SHIELD itself, or wish to write a
plugin to extend the capabilities of your SHIELD installation, head on over
to the [SHIELD Developer Documentation]($docs/dev).


What is SHIELD?
---------------

SHIELD is a _data protection_ solution.  It is designed to run scheduled
tasks to backup your important data systems to off-site cloud storage
solutions, and facilitate the restoration of backup archives in the event of
outages or data loss.

SHIELD supports lots of different data systems, through its flexible and
modular plugin architecture.  We currently support:

  - **PostgreSQL** databases, via the [`postgres` plugin][postgres-plugin].
  - **MySQL** / **MariaDB**, via the [`mysql` plugin][mysql-plugin], or the
    [`xtrabackup` plugin][xtrabackup-plugin].
  - **Redis** key-value store, via the [`redis` plugin][redis-plugin].
  - ... and [many more][target-plugins].


[postgres-plugin]:   $docs/ops/plugins/postgres
[mysql-plugin]:      $docs/ops/plugins/mysql
[xtrabackup-plugin]: $docs/ops/plugins/xtrabackup
[redis-plugin]:      $docs/ops/plugins/redis
[target-plugins]:    $docs/ops/plugins#target-plugins

Cloud Storage systems are likewise pluggable.  Out of the box, SHIELD
supports:

  - **Amazon S3** (and S3 work-alikes) via the [`s3` plugin][s3-plugin].
  - **Microsoft Azure Blobstore** via the [`azure` plugin][azure-plugin].
  - **GCP Blobstore** via the [`google` plugin][google-plugin].
  - On-premise **WebDAV** endpoints, via the [`webdav` plugin][webdav-plugin].
  - ... and [several others][store-plugins].

[s3-plugin]:     $docs/ops/plugins/s3
[azure-plugin]:  $docs/ops/plugins/azure
[google-plugin]: $docs/ops/plugins/google
[webdav-plugin]: $docs/ops/plugins/webdav
[store-plugins]: $docs/ops/plugins/#store-plugins

SHIELD is a distributed system.  The SHIELD _core_ leverages a network of
_agents_ to do the heavy lifting of data backup and restore.  When you
deploy SHIELD into your infrastructure, yet get to choose how many agents
you want to provision, and where in the network they sit.

// FIXME: link to more in-depth docs on the architecture of SHIELD

_Multi-tenancy_ is baked right into SHIELD via a robust role-based access
control (RBAC) system in place to help isolate different subsets of users
from one another.  People in one tenant are unable to see configurations
made by people in another tenant.  This allows a single SHIELD to support
multiple, independent teams.

// FIXME: link to more in-depth docs on MT and RBAC

SHIELD also supports a sophisticated authentication system.  You can hook
up to your external Cloud Foundry UAA server, a BOSH UAA instance, or even
Github (both public and on-premise).  As users log in via their external
credentials, SHIELD will automatically create the necessary tenants and
assign roles based on the SHIELD configuration.

// FIXME: link to more in-depth docs on how Authn works

We believe strongly in encryption.  Whenever SHIELD communicates across the
network, it does so over encrypted channels (SSH, TLS/HTTPS), with endpoint
identity verification (host keys, mutual TLS, etc.).  All backup archives
are encrypted with unique key material, to ensure that data at-rest is also
resistent to snooping and tampering.

// FIXME: link to more in-depth docs on how SHIELD encryption works

Installation
------------

Before you can begin to use SHIELD to protect your important data, you're
going to need to install it.  You have two options: via [BOSH][bosh]
(ideal for Cloud Foundry users), or via [Docker][docker].

[bosh]:   https://bosh.io
[docker]: https://docker.com

### Planning Your Installation

Before you start installing the software, it's worthwhile to take a step
back and plan out your installation

#### Network Topology

SHIELD operates best on flat network topologies, without <abbr>NAT</abbr>
devices or HTTP(S) proxy services.

//![A Flat Network Topology]($docs/ops/manual/flat-topology.png)

SHIELD requires _mutual network visibility_ between the core and all
cooperating agents.  Each agent issues a small HTTP request to the core,
to inform the core that it is alive, and ready to be inventoried.  This is
called the _registration ping_.  For each registration ping received, the
core records the name and port given and the remote address of the
connecting TCP socket.  At some later time, the SHIELD core will initiate an
SSH connection to the recorded agent IP address, and gather agent
information.

Because of this, NAT devices tend to confound SHIELD.  The registration ping
originates (at the TCP level) from the NAT gateway, not the host running the
SHIELD agent software.  When the core attempts to connect back to the agent,
it initiates a connection to the _NAT device_ on the agent port, which
generally fails.

Flat networks with HTTP(S) proxies are not impossible, but they can be
unruly.  When configuring proxy clients (via `http_proxy` / `https_proxy`,
`no_proxy` environment variables, or similar mechanisms), you will want to
be especially cognizant of the HTTP(S) connections needed by the SHIELD
software itself.  Often, these connections will need to bypass am
Internet-bound proxy (i.e. one in a DMZ) in order to function.

//![A Proxied Network Topology]($docs/ops/manual/proxied-topology.png)

Likewise, if your cloud storage solution is to be dealt with over HTTP(S),
you will need to make sure that either your proxy server can contact it on
behalf of each SHIELD agent, or that each agent blacklists the domains
and/or IP addresses of the storage endpoints in something like `no_proxy`.

#### Where to Colocate SHIELD Agents

Depending on the data systems you wish to backup, and their configuration
with respect to access control, you may be able to get away without
colocating _any_ SHIELD agents in your infrastructure.

There are really only two reasons for colocating a SHIELD agent on a data
system installation: plugin requirements and host access control
configuration.

Most SHIELD plugins stream their data through, without relying on any
temporary local storage.  This removes a throughput bottleneck (the disk),
as well as a capacity concern (how much temporary space do you need?).  Some
plugins, however, require local disk.  If the target system doesn't require
local access (more on that in a moment), you may want to spin up some
machines with large ephemeral disks just to handle these backup / restore
operations.  The conversation for backing up data then looks like this:

![A Dedicated SHIELD Agent]($docs/ops/manual/dedicated-shield-agent.png)

Some plugins absolutely cannot be executed across the network.  The
[Filesystem Plugin][fs], for example, can only deal with files on the local
filesystem (networked filesystems notwithstanding).  Therefore, if you need
to back up files on a host, you will need to deploy a SHIELD agent to run on
that host.

[fs]: $docs/ops/plugins/fs


### Using BOSH

[BOSH][bosh] is a cloud-agnostic deployment and orchestration tool that
excels at lifecycle management of software at all scales.  SHIELD has a
_BOSH release_ that can be used to deploy both the SHIELD core, and SHIELD
agents into new and existing BOSH deployments.

If you're already using BOSH (for example, if you are deploying Cloud
Foundry), adding SHIELD into your infrastructure should be easy.  If you are
still looking for a great release engineering framework, you can get your
feet wet with a SHIELD deployment or three.

The SHIELD BOSH release can be found [on Github][release].

[release]: https://github.com/starkandwayne/shield-boshrelease/releases

#### Deploying the SHIELD Core

Usually, the SHIELD core is a standalone, self-contained deployment.  To
deploy, you'll need to find or create a deployment manifest.  A good
starting point can be found [here][manifest.yml].

//[manifest.yml]: https://github.com/starkandwayne/shield-boshrelease/blob/master/manifests/shield.yml
[manifest.yml]: $docs/ops/manual/shield.yml

Save that file locally, as `shield.yml`, and then run:

    $ bosh -e my-bosh deploy \
        -d shield \
        -v static_ip=:::192.0.2.5::: \
        -v domain=:::shield.example.com::: \
        shield.yml

Replace `domain` with the FQDN of your SHIELD management console, and
`192.0.2.5` with a static IP that you want to deploy SHIELD on.  You may
need to consult your BOSH [cloud-config][bosh-cc] to find a suitable IP in
the `default` network.  Optionally, you may modify the deployment manifest
to specify a different network.

[bosh-cc]: https://bosh.io/docs/cloud-config

**NOTE**: The provided deployment manifest assumes that your BOSH director
has been deployed with a _config-server_ that can generate the necessary
certificates and keys for securing SHIELD's communications.  If that is not
the case, you will need to provide additional command-line options to the
`bosh deploy` command to store the generated credentials locally.  See
[the BOSH documentation][bosh-vars] for more information.

[bosh-vars]: https://bosh.io/docs/cli-int/#vars-store

Once BOSH has finished deploying SHIELD, you should be able to access the
SHIELD management console at [https://$IP](#).  The default login will be
`admin` (username) and `password` (password).

#### Deploying SHIELD Agents

If you need to colocate agents on other BOSH deployments, you have a few
options.  The fastest method is to modify those deployment manifests to
include the `shield-agent` job in the appropriate BOSH instance groups, like
this:

    instance_groups:
      - name: some-database
        jobs:
          # ... other jobs ....
          :::- name:    shield-agent
            release: shield:::

        # ... rest of configuration ...

This can get out of hand fast.  A more elegant solution is to use [BOSH
runtime configs][bosh-rcs] and _inject_ the SHIELD agent job into other
deployments without mucking about with their deployment manifests.

[bosh-rcs]: https://bosh.io/docs/runtime-config

Here's a working runtime config:

    ---
    releases:
      - name:    shield
        version: 8.0.8

    addons:
      - name: shield
        jobs:
          :::- name:    shield-agent
            release: shield:::

To use this, update your existing runtime-config:

    $ bosh update-runtime-config addons.yml

Then, `bosh deploy` your pre-existing manifests, without changing them.
For more information, including how to limit the `shield` addon to just
specific deployments / VMs, read the
[BOSH runtime-configs documentation][bosh-rcs].


### Using Docker

#### The SHIELD Core Image

TBD

#### The SHIELD Agent Standalone Image

TBD

#### Embedding the SHIELD Agent

TBD


### Configuration Reference

This section contains detailed descriptions of all configuration options for
the SHIELD core, and SHIELD agents.

#### SHIELD Core Configuration File Reference

The SHIELD core configuration file is a YAML file, read at startup by the
`shieldd` binary.

Here's an example configuration file:

    ---
    # comments start with the octothorpe (#) and continue
    # until the end of the line

    api:
      env: Production
      color: green
      bind: 127.0.0.1:80
      session:
        timeout: 300         # (hours) - about 2 weeks
      failsafe:
        username: admin
        password: super-sekrit-password

    scheduler:
      threads: 20

    limit:
      retention:
        min: 4
        max: 90

Here's the full set of configuration directives (in dot-notation)
that are currently supported:

- **api.bind** - The IP address and TCP port that the SHIELD core daemon
  should bind to and listen on for incoming API (HTTP) requests.  Defaults
  to `*:8888`.  `*` is interpreted to mean _all interfaces_.

- **api.color** - You can color code your SHIELD Web User Interfaces!  Set a hex
  value here (i.e. `##003300`) or other CSS-compatible color identifier, and
  the web UI will use it to colorize the environment name.

- **api.env** - An name for the environment, that SHIELD will pass
  through to clients accessing its API and web management console.  This can
  be useful for differentiating your staging SHIELD from your production
  SHIELD.  By default, no environment is set.

- **api.failsafe.username** - When the SHIELD core starts up, it
  checks the local users table.  If it is empty (there are no
  local users), it creates a _failsafe_ account, using these
  parameters.  This is designed to assist in a safe and secure
  bootstrap.

- **api.failsafe.password** - The cleartext password to assign the
  failsafe user, upon creation.  You can change this later,
  without needing to reconfigure or re-deploy SHIELD.

- **api.motd** - A (hopefully) short message to display to operators on the
  login screen.  You can use this for compliance messages, important
  notices, an explanation of which authentication method people should use,
  who to contact for help, etc.  By default, there is no MOTD.

- **api.session.clear-on-boot** - Whether or not to clear
  interactive sessions from the database when the SHIELD core
  reboots.  This is set to `yes` by default, which will log
  everyone out when the core process restarts.

- **api.session.timeout** - How long (in hours) before idle
  authenticated sessions are invalidated.  Defaults to 720 hours
  (about a month).

- **auth** - A list of non-local authentication backends.

- **auth[].backend** - What backend to use for this authentication
  provider.  The backend determines how authentication is carried
  out, what external systems are involved, etc.

  The following backends are supported:

  - **github** - GitHub, either public (github.com) or private
    (Github Enterprise).  Tenant membership is mapped based on
    GitHub organization / team membership.

  - **uaa** - Cloud Foundry UAA, to allow Cloud Foundry users to
    log into SHIELD without needing another account.  Tenant
    membership is mapped based on scim rights.

- **auth[].identifer** - A unique (and unchanging) internal
  identifier for this authentication provider.

- **auth[].name** - The human-readable name for this
  authentication providers, to be displayed in command-line and
  web UI login flows.

- **auth[].properties** - A set of backend-specific configuration
  items.

- **cipher** - Which encryption algorithm and chaining mode to use
  for encrypting backup archives.  Supported values are:

  - `aes256-ctr` - 256-bit AES, in Counter CBC mode.

  We plan to introduce more types as the need arises.

  Each backup archive tracks which encryption type was in force when it was
  taken, to allow operators to change this value without rendering previous
  backup archives unusable.

- **data-dir** - The absolute path to the directory where SHIELD will
  store all of its persistent data.  Important files stored here include:

  - `$datadir/shield.db` - The SHIELD metadata database.
  - `$datadir/vault/*` - The encrypted files that back the vault.
  - `$datadir/vault.crypt` - The encrypted file which stores the seal
    keys to the SHIELD Vault.  This file is encrypted with the
    SHIELD master password.
  - `$datadir/bootstrap.log` - A log of what occurred during a SHIELD
    _from-nothing_ recovery.

- **debug** - Whether or not to enable verbose debug logging.  This is a
  boolean, and it defaults to `no`, which is a sane choice for any
  production or staging environment.  Debug logging is verbose, and very
  low-level.  It is of primary value to SHIELD developers.

- **fabrics** - A list of SHIELD Agent Fabric configurations.

- **fabrics[].name** - The name of this fabric.  For the Legacy
  SSH fabric, this must be `legacy`.

- **fabrics[].ssh-key**  - The literal, PEM-encoded RSA private
  key that the SHIELD core should use when connecting to remote
  SHIELD agent processes.

  This is **required** for the Legacy SSH fabric.

- **limit.retention.max** - The maximum number of days that any
  backup archive may be retained for.  This can be useful to
  ensure that tenants don't overrun storage with excessively long
  retention periods.

- **limit.retention.min** - The minimum number of days that all
  backup archives must be retained for.  This allows SHIELD
  operators to enforce organization-wide compliance for archive
  retention.  Defaults to 1 (day).

- **scheduler.fast-loop** - The frequency, in seconds, of the
  SHIELD scheduler's "fast loop."  On every iteration of the fast
  loop, SHIELD will schedule backup jobs that ought to run, execute
  pending tasks (if it has workers available), and handle inbound
  agent registration pings.

  By default, the fast loop executes once a second.  Unles you have an
  urgent need otherwise, you shouldn't change this.

- **scheduler.slow-loop** - The frequency, in seconds, of the
  SHIELD scheduler's "slow loop."  The slow loop handles
  administrative tasks for the SHIELD core, including archive
  expiration and purgation, session clearing, data analytics, and
  cloud storage testing>

  By default, the slow loop exucutes once every 300 seconds (5 minutes).
  Turning up the frequency will result in higher load on external cloud
  storage systems.  Decreasing the frequency will cause expired archives to
  remain in cloud storage for longer.

- **scheduler.threads** - How many worker threads should the
  SHIELD core spin.  This defaults to `2`, but you should increase
  the 1.5 times the number of concurrent backup tasks you expect to
  see, at peak.

  Low worker counts can cause the SHIELD scheduler to "stall out"
  and not execute scheduled tasks in a timely fashion.  The 1.5x
  multiplier accounts for purge operations, cloud storage tests, and
  other background tests.

- **vault.address** - The URL of the SHIELD Vault.  This should almost
  always be `https://127.0.0.1:8200`.  If you are using the BOSH release,
  this cannot be configured.

- **vault.ca** - The path to the X.509 Certificate Authority certificate,
  PEM-encoded, for validating the Vault certificate.  If you are using the
  BOSH release, this cannot be configured (nor does it need to be).

- **web-root** - The root path to the SHIELD web management UI assets.
  Defaults to the relative path `web`, which is probably not what you want.


#### SHIELD Agent Configuration File Reference

The SHIELD agent configuration file is a YAML file, read at startup by the
`shield-agent` binary.

- **name** (required) - The name of this agent, for registration with the
  SHIELD core.  This name will appear in web and CLI interfaces, to people
  configuring backup jobs, and should describe the role this agent
  installation plays in the overall topology.

- **authorized\_keys\_file** (required) - The path to an SSH _authorized
  keys_ file, which should contain the public component of the agent private
  key that the SHIELD core will use to authenticate to the agent for remote
  orchestration.

- **listen\_address** - The IP address and TCP port that the SHIELD agent
  should bind to and listen on for incoming orchestration (via SSH).
  Defaults to `*:5444`.  `*` is interpreted to mean _all interfaces_.

- **plugin\_paths** (required) - A YAML list of paths that the agent will
  use when attempting to resolve plugin names to binaries.  This is kind of
  like the canonical UNIX `$PATH` environment variable, except it does not
  apply to any programs that the plugins themselves attempt to execute.

  You should list all of your plugin binary directories here.

- **registration** - This subsection governs how this agent will register
  with its SHIELD core.  While technically optional, registration is highly
  recommended, from an ease-of-use standpoint.

  The following keys exist underneath `registration:`:

  - **url** - The HTTP(S) URL of the upstream SHIELD core.  This will
    normally be something like `https://$ip_or_hostname/`.

  - **interval** - How often (in seconds) should the agent ping the SHIELD
    core and provide reigstration details.  The SHIELD core determines when
    it _validates_ agent registrations and extracts metadata information, so
    this setting cannot be used to increase the frequency of such updates.

  - **shield\_ca\_cert** - Path to a file containing the PEM-encoded CA
    certificate that issued the SHIELD core's X.509 TLS certificate.  This
    allows operators to validate self-signed certificates, or custom,
    in-house CA-issued certificates.

    This has no effect of `skip_verify` is set to true.

  - **skip\_verify** - Whether or not to disable verification of the SHIELD
    core X.509 TLS certificate.  This defaults to _false_, since certificate
    verification is generally A Good Thing &trade;


Using SHIELD
------------

SHIELD features a beautiful web user interface and a robust command-line
interface.  We like to think of the web UI as providing more visibility into
the configuration of SHIELD, while the CLI provides more flexibility in
terms of automation.

### The Web UI

You can access the web UI by pointing your browser at the IP address of your
SHIELD core installation.  SHIELD forces all HTTP traffic over TLS, via port
443, for security reasons.

#### Logging In

Before you can interact with SHIELD, you must log in.

![The SHIELD Web UI Login Screen]($docs/ops/manual/ui/login-screen.png)

On the right is the login form for local authentication.  On the left is a
list of the configured _authentication providers_.  These allow SHIELD
administrators to integrate SHIELD authentication with 3rd-party, external
identity systems like Github, or Cloud Foundry UAA.

Note: You may not have any authentication providers listed.

#### The Heads-up Display

All logged in?  Great!

At the top of the screen, you should see the _heads-up display_:

![The SHIELD Heads-Up Display]($docs/ops/manual/ui/hud.png)

To the left is identifying information about this SHIELD core, including the
configured SHIELD environment name, the IP address and/or fully-qualified
domain, and the version of SHIELD.

The first pane summarizes the overall health of SHIELD and the current
tenant's configuration.

  - **SHIELD is ...** - Reports the current status of the SHIELD API.
    If all is well, this will say _SHIELD is up_, in a reassuring green hue.
    If the SHIELD core is not responding to API calls, this will say _SHIELD
    is DOWN_, in red.  Sometimes, it may report that the SHIELD is locked,
    in which case an administrator needs to intervene to unlock it.

  - **Cloud Storage is ...** - Reports the health of all global cloud
    storage systems, as well as the health of all cloud storage specific to
    the currently selected tenant.

  - **Jobs are ...** - Reports the status of all backup jobs for the current
    tenant.  It considers only the most recent execution of each job,
    whether it was scheduled or run manually (ad hoc).

The second pane, titled _Data Protection Summary_, provides some numbers for your
consideration.  All of these are per-tenant.

  - **Scheduled Backup Jobs** - How many total jobs are scheduled to run.

  - **Backup Archives** - How many backup archives exist.

  - **Cloud Storage Used** - How much of cloud storage is being used by the
    backup archives for this tenant's jobs.

  - **Daily Storage Increase** - A simple linear projection of the amount of
    additional cloud storage that will be used, each day, given the current
    schedules, retention policies, and archive sizes.

The heads-up display is partially dependent on the current tenant, so if you
switch to a different tenant, you might get different numbers / statuses.

#### The Task Sidebar

To the left of the screen is a sidebar with links to the common tasks you
may want to perform:

  - Run an ad hoc backup
  - Restore data from a backup
  - Configure a new backup job

#### The Top Bar

TBD

#### The Navigation Bar

The black navigation bar (immediately under the heads-up display) will stick
to the top of the viewport as you scroll.  It provides top-level navigation,
including:

- **Systems** - Your data systems are the things that SHIELD protects, by
  making copies of the important data contained within them, on a scheduled
  and recurring basis.  This page lets you review and manage those systems.

- **Storage** - Cloud storage is where SHIELD keeps the backup copies of
  your data.  You can configure however many storage systems you want, in
  whatever configuration you deem appropriate.  This page lets you access
  both global (shared) cloud storage systems, and those specific to your
  tenant.

- **Admin** - This one is for SHIELD administrators only.  It provides
  access to a host of administrative functions like tenant and user
  management, global cloud storage management, etc.


### The CLI

If you prefer life in the terminal, you're in luck &mdash; SHIELD
has a command-line interface packed with just as much
functionality as the web interface.  To get started with the CLI,
you first must configure access to the SHIELD core by identifying
the API endpoint and logging in:

    $ shield api https://:::shield-ip::: my-shield

    $ shield -c my-shield login
    Username: admin
    Password:
    logged in successfully!

To see what more the CLI can do, check out the following helpful
commands:

    $ shield -h
    $ shield commands

Note: the remainder of this guide will focus on the web user
interface.  Terminal fans are encouraged to explore the SHIELD CLI
to see how those same concepts can be applied in a textual world.



Configuring Backups
-------------------


TBD

### Wizard Walkthrough

TBD

### The Systems Page

TBD

### Adding a second schedule

TBD

Running Backups / Restores
--------------------------

The primary point of running SHIELD is to run regularly-scheduled
backup jobs, and then (when necessary) restore backups to their
original data system.

### Ad hoc vs. Scheduled

SHIELD can run backup jobs in one of two ways.

**Ad hoc** backup jobs are executed in one-off scenarios, at the
direction of a SHIELD operator.

**Scheduled** backup jobs are run at specific, cyclical times.
Operators can specify that backups should be taken hourly, daily,
weekly, or monthly.  Inside SHIELD, the _scheduler_ keeps track of
the next scheduled run of each job, and executes accordingly.

For maximum value, you will probably want to schedule most of your
backups.  The two modes of operation are not mutually exclusive;
you can trigger an ad hoc run of a scheduled job.  This comes in
handy when carrying out maintenance or change operations against a
system and you want the freshest _pre-change_ backup archives as
you can get.

### The Ad hoc backup Wizard

TBD

### The Timeline View

TBD

### Annotating Tasks

TBD

### Restoring from the Timeline Page

TBD

### The Restore Wizard

TBD

Cloud Storage
-------------

TBD

### How SHIELD Uses Cloud Storage

TBD

### Retention Policies

TBD

### How the HUD interacts

TBD

### Storage Thresholds

TBD

### The Storage Display Page

TBD

### Shared Storage

TBD

Multi-Tenancy
-------------

TBD

### What is a Tenant?

TBD

### Switching Tenants

TBD

### Role Assignments (and what they mean)

TBD

### Authentication Providers and Tenants

TBD

#### UAA

TBD

#### Github

TBD

### The default tenant

TBD

Encryption
----------

TBD

### What is Encryption

TBD

### Why do I Care?

TBD

### How does SHIELD use encryption?

TBD

### At-rest vs. in-flight encryption

TBD

Administration
--------------

All Administrative tasks are done from the admin panel 

![The Shield Admin Panel](admin-panel.png)

### Initializing A SHIELD Core

TBD

### The Master Password

TBD

### The Administrative Backend

TBD

#### Tenants

Under `Tenants` admins have the ability to see all current teneants as well as create new tenants.
Users can be invited to tenants by editing the tenant or during tenant creation. 

![Tenants](tenant-panel.png)

#### Shared Storage

Shared storage can be found under `Global Storage Systems`, from this page you can view all existing
global stores as well as create new ones. Shared or global storage can be configured for use across
multiple tenants.

![Global Storage](storage-panel.png)


#### Retention Policy Templates

TBD

#### Managing Agents

Information on Shield Agents can be found under `Agents of Shield`. This can be helpful for keeping
inventory on agents throughout your deployments or to resync any disconnected agents. 

![Shield Agents](agents-panel.png)

#### Authentication Providers

You can view the configuration of your shield authentication providers under `Authemtication Providers`.
Please note that configuring an additonal auth provider is done from deploying shield itself.

![Authentication](auth-panel.png)

#### Local User Management

Shield users can be managed under `Local User Management`. Here admins can view a list of all Shield users,
edit permissions of users, and onboard new users.

![Users](user-panel.png)

#### Rekeying SHIELD

If you wish to Change your shield master password, you can find this option under `Rekeying SHIELD`

**Note** You need your current master password to complete this.

![Panel](rekey-panel.png)

#### Session Management

Admins can manage user sessions under `Manage User Sessions`. Here You can get information on all current sessons
and expire session tokens if desired.

![Sessions](session-panel.png)

How Do I Backup _X_?
--------------------

### SHIELD Itself

In a disaster recovery situation, it's important to get shield up and running to begin recovering your other data systems
as soon as possible. It requires backups of SHIELD itself.

#### How do SHIELD backups work? 

The SHIELD state exists as 3 entities found in /var/vcap/store/shield (for a shield deployed by bosh).
They Include:
  - The SHIELD database which exists as a sqlite database name shield.db
  - The vault.crpyt file
  - The vault directory

We use the file system plugin to backup these 3 items and store them as our SHIELD backup. When using the fs plugin to backup
shield make sure `Strict Mode` is unchecked. We do this to prevent failures with missing files which can happen with the presence
or absence of the `shield.db-journal`.

#### The Problem With Encryption

SHIELD backups *must* be taken with fixed key encryption. In this situation of restoring shield, we cannot use generated
encryption keys because there is no way to restore our vault with keys that exist in that sealed vault. This is why it's
important to keep the fixed key generated when you initialize SHIELD in a safe and retrievable place.  

#### Configuring the `fs` Plugin

Lets walk through creating a SHIELD backup from the shield webui.

#### For the properties make sure you're targeting the agent on the SHIELD box and the correct path.

  ![Recover](shield-recover2.png)

#### Make sure to de-select `Randomize Encryption Keys`

  ![Recover](shield-recover4.png)

#### Lets double check our configuration is correct

  ![Recover](shield-recover5.png)

#### "Normal-mode" Restores

TBD

### Cloud Foundry UAA

### Cloud Foundry CCDB

### BOSH

Monitoring SHIELD
-----------------

TBD

### Using the HUD

TBD

### API Access for Monitoring

TBD

### Metrics of interest

TBD

### Log messages to watch

TBD

Glossary
--------
Shield has a number of terms and abstractions that are important for an operator to know when working with the tool.

- **Shield Core**: The shield core servers as the main deployed shield box that includes the shield daemon and api.

- **Shield Agents**: A shield agent is software that is co-loacted on a deployment you wish to back up that leverages plugins to back up given data system

- **Tenants**: Shield provides tenancy to allow the seperation of teams/environments by allowing tenants to be created. Under a tenant users can be assigned 
               and given roles, stores can be created, and indivdual systems can be backed up.

- **Targets**: A target is the name of a configured data system. They can be under the data systems tab.
