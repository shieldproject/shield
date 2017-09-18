SHIELD Multi-Tenancy and Role-Based Access Control
==================================================

Oftentimes, people wish to deploy a single SHIELD instance and
use it for multiple internal teams.  Currently (as of 0.10.x),
SHIELD does not support this.

This document lays out a design and a plan to implement that would
allow that use case to play out in production environments.

What Is Tenancy?
----------------

A Tenant is a single group that defines the context for
interaction with resources in a SHIELD configuration.  All
retention policies, jobs, backup targets, storage endpoints and
archives belong to a single tenant.

For example, a SHIELD may serve the infrastructure team and two
application development teams.  Each of these groups needs to be
able to specify their backup job configuration, how long to keep
the archives (retention policies), what to backup (targets and
jobs), and where to store the archives (storage).

Tenancy exists to keep these three groups isolated from one
another, for the following purposes:

  - To insulate change
  - To protect confidentiality of systems

By _insulating change_ we mean that one team is free to
reconfigure how long they retain their backups, without affecting
the other teams adversely.  The infrastructure team, for example, may
need to keep several months of platform backups, so long-term for
them might be 90d.  To the app teams, long-term might be a week.

Therefore, each tenant gets its own set of retention policies, and
they can determine what "long-term" means to them.

Confidentiality is protected on two fronts.  First, and foremost,
the archives created as a result of performing backups must only
be available to that tenant.  If the infrastructure team regularly
backs up a centralized account store (which contains personnel
data), the application teams should be prohibited from restoring
those archives to a system they control.  Otherwise, personal
information will be at risk.

Therefore, each archive is assigned to the tenant who owns the job
that created it, and tenants are prevented from accessing or
viewing another tenant's backups.

The other aspect of confidentiality relates to target and storage
configuration.  These endpoint configurations contain sensitive
material, like S3 access keys, database credentials, and API
tokens.  Viewing these outside the scope of the tenant who owns
them is a serious breach of the trust model.

Therefore, each tenant creates and manages its own target and
storage configurations, and tenants are prevented from accessing
or viewing another tenant's configuration.

The upshot of all of this is that each tenant operates inside the
SHIELD instance as if they were the only tenant present; they
cannot see or interact with the other tenants.

Global Sharing
--------------

Sometimes, sharing is desirable.  A company, for example, may want
to utilize a single S3 bucket for storing all of their backup
archives, yet still grant each team their own tenant space inside
of the corporate SHIELD.

The reasoning behind this particular scenario is a bit nuanced, so
let's consider it in more detail.

With the coming [encryption feature][enc], there is little benefit
in accessing the backup archives directly.  Indeed, without the
keys to the encrypted archives, access to the store is of little
value.

Therefore, it is entirely possible that all tenants could share a
single storage endpoint.  SHIELD would ensure that no two archives
reside at the same position in that storage backend (which it has
to do already, today).  If the SHIELD users are not given the
credentials to the shared storage, they are effectively segmented
per SHIELD tenant boundaries, but the company doesn't have to spin
up new buckets or storage locations for each tenant they create.

Even without the encryption piece, we are seeing clients do this
today; storing unrelated teams backup archvies on the same S3
bucket or internal NFS mount.

To continue supporting this practice, SHIELD needs to understand
the concept of _global_ sharing.  If something is _globally
shared_, it can be used by all tenants, but can only be managed by
SHIELD site administrators.  Tenants will be prohibited from
viewing the configuration details of the thing shared, to prevent
them from gaining out-of-band access to the storage backend.

Other things (aside from storage endpoints) at first seem to
benefit from the same _global sharing_: primarily, retention
policies.  However, instead of following that line of thinking,
let's consider an alternate approach: _templating_.

Retention policies are fairly flat resources - they are entirely
self-contained, straightforward and simple.  There is not much
benefit to be had by sharing them, and there is a large downside:
it breaks the _change insulation_ of tenancy.

Consider what happens if several tenants all rely on the
"Long-term" retention policy, originally defined as 90 days.
What happens when one team still needs longer-term storage, but is
generating so many archives that they wish to reduce the 90 days
to 30 days?  As a SHIELD operator, you have two choices:

  1. Modify the shared "Long-term" policy and change retention
     from 90 days to 30.
  2. Create a new policy, for 30 days of retention, have the tenant
     reconfigure all of their jobs, and then wait for the archives
     that are pegged at "keep 90 days" age out.

Clearly, the first solution is a non-starter: modification to the
shared policy will affect *every* tenant using that policy.

The second solution is the correct one, given _global sharing_,
but is far from ideal.

Instead, what if SHIELD gave every tenant a copy of the
globally-defined retention policies?  Each tenant would then have
full control over their scheduling and retention policies.  If a
tenant wants to modify them, they can, and it will only affect
them.

This is called _templating_.  It only applies to retention
policies.  After a tenant is created, any changes to the global
retention policies will only affect future tenants, as they are
created.  In the real world, this is rarely a problem, since
retention policy management tasks are few and far between.

Lastly, we should talk about the things that are never shared:

  - **Targets** - The designers can think of no situation in which
    multiple tenants need to perform backup / restore operations
    against a shared data system.
  - **Jobs** - Without sharing targets, there isn't much point in
    sharing jobs.


Roles
-----

Some times, we may want to grant someone access to our SHIELD
tenant, but only on restricted terms, with a subset of "full"
access.

For example, we may want to have a NOC team be able to re-run
failing backup jobs, but not be allowed to change any
configuration, or see the credentials stored in target / storage
endpoints.  This is where roles come into play.

A **Right** is a token that governs one single aspect of the
system.  The ability to view backup archives would be a right; so
would the ability to run an ad hoc backup job.

A **Role** represents a given level of access to either a tenant
("Tenant Roles") or the system itself ("System Roles").  These
roles are defined by the SHIELD software itself.  They are:

- **Site Administrator** - super user
- **Site Manager** - able to manage tenants and assignments
- **Site Engineer** - site-level configuration management (i.e.
    shared cloud storage definitions, retention policy templates,
    etc.)
- **Tenant Administrator** - manages tenant membership and roles
- **Tenant Engineer** - manages configuration of tenant systems
    and backup jobs.
- **Tenant Operator** - day-to-day operation; run / restore /
    pause / unpause / view access.

Each user is assigned one or more roles, for each tenant that they
have access to.


Authentication Backends
-----------------------

Most operators work in an environment with existing authentication
systems, be they Active Directory / LDAP, Github authentication,
some form of UAA, etc.  It would be desirable for SHIELD to align
with those authentication systems, rather than provide yet another
set of credentials for operators and their customers to manage.

To that end, SHIELD (as of 0.10.8) supports Github and CF UAA
(although there are [issues][cf-uaa] with that) OAuth2 systems for
authentication.

These systems will continue to be supported, but will go through
some changes to support role-based access control.

Primarily, SHIELD will be able to support multiple authentication
backends concurrently.  This does not make much sense with the
big OAuth2 systems like Github and CF/UAA, but it starts to make
sense when we add API Key and Local Authentication as
authentication backends.

SHIELD administrators may opt to keep local authentication for
root-level access to their SHIELD environments, as a sort of
emergency avenue for performing backups.  This allows SHIELD to
safely backup (and restore!) the authentication data itself,
without running into primacy problems.

What follows are rough sketches of how each proposed
authentication backend interacts with Roles &amp; Rights, and how
the SHIELD UI / CLI must adapt to accommodate.

### Local Authentication Backend

Local Authentication is completely independent of external
systems.  SHIELD itself maintains a list of user accounts, data
for verifying passwords (bcrypt hashes or the like), and their
tenant/role assignments.

This is the simplest backend conceptually, but requires the most
change to the SHIELD codebase and UI/CLI.  The UI and CLI will
need new features (screens and commands) that allow SHIELD site
administrators to create new accounts, disable and remove
accounts, assign tenants and roles to accounts, etc.

In the interest of shipping the RBAC feature, we may focus on the
web interface and forego the CLI in the first round.

### Github Authentication Backend

When SHIELD verifies the bearer token, Github returns a list
of the organizations and groups the user belongs to.  The
configuration for the Github Authentication Backend maps these
org/group combinations to tenant/roles.  The SHIELD site
administrator is responsible for maintaining this mapping.

It is not possible to _override_ the org/team &rarr; tenant/role
mapping to grant or revoke tenant/role assignments manually.

The SHIELD UI and CLI will need new screens and commands to allow
SHIELD site administrators to configure the Github integration
(URL, application client/secret), and set up the role mapping
rules.

In the interest of shipping the RBAC feature, we may focus on the
web interface and forego the CLI in the first round.

### UAA Authentication Backend

When SHIELD verifies the bearer token, UAA returns a list of
groups the user belongs to.  The configuration for the UAA
Authentication Backend maps these groups to tenant/role
combinations.  The SHIELD site administrator is responsible for
maintaining this mapping.

It is not possible to _override_ the group &rarr; tenant/role
mapping to grant or revoke tenant/role assignments manually.

The SHIELD UI and CLI will need new screens and commands to allow
SHIELD site administrators to configure the UAA integration
(URL, application client/secret), and set up the role mapping
rules.

In the interest of shipping the RBAC feature, we may focus on the
web interface and forego the CLI in the first round.

### API Key Authentication Backend

Right now, API keys are used by supplying a custom header,
`X-Shield-Token` in the request.  The API code verifies the key
against a list of keys.

With the introduction of multi-tenancy, API keys need overhauled;
API keys must be tied to a set of tenants and roles, just like
regular accounts.  The biggest change is that API keys will need
to be treated as "live" configuration, moved out of the SHIELD
daemon configuration, and into the database proper.

The SHIELD UI and CLI will need new screens and commands to allow
SHIELD site administrators to generate, modify and revoke API
keys, map them to tenant/roles, and annotate them.

In the interest of shipping the RBAC feature, we may focus on the
web interface and forego the CLI in the first round.


Auditing
--------

With tenants and roles, SHIELD is equipped to generate audit logs,
odetailing who carried out what transactions, when, as well as who
was denied access to perform a given task.

The key details of an audit log record are:

  - Who
    - Username / API Key
    - Authentication Backend
    - Affected Tenant (or global)
    - Remote IP
  - What
    - The Task
    - Involved Resources (if any)
  - When
  - Why
    - Was the user acting as an **Administrator**?
    - If not, what role granted the privilege


[enc]: https://github.com/starkandwayne/shield/blob/master/docs/encryption.md
