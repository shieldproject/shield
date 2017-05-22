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
schedules, retention policies, jobs, backup targets, storage
endpoints and archives belong to a single tenant.

For example, a SHIELD may serve the infrastructure team and two
application development teams.  Each of these groups needs to be
able to specify when their backups run (schedules), how long to
keep the archives (retention policies), what to backup (targets
and jobs), and where to store the archives (storage).

Tenancy exists to keep these three groups isolated from one
another, for the following purposes:

  - To insulate change
  - To protect confidentiality of systems

By _insulating change_ we mean that one team is free to
reconfigure when their daily backups run, without affecting the
other teams adversely.  The infrastructure team, for example, may
have a lull in the late evening time frame (say, 10pm - midnight)
during which they wish to perform backups if critical systems.
This may not mesh well with a 3am-6am change window for the
application teams.

Therefore, each tenant gets its own set of schedules, and they can
determine when "daily" is for them.

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
benefit from the same _global sharing_: primarily, schedules and
retention policies.  However, instead of following that line of
thinking, let's consider an alternate approach: _templating_.

Schedules and retention policies are fairly flat resources - they
are entirely self-contained, straightforward and simple.  There is
not much benefit to be had by sharing them, and there is a large
downside: it breaks the _change insulation_ of tenancy.

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
globally-defined retention policies and schedules?  Each tenant
would then have full control over their scheduling and retention
policies.  If a tenant wants to modify them, they can, and it will
only affect them.

This is called _templating_.  It only applies to schedules and
retention policies.  After a tenant is created, any changes to the
global schedules and retention policies will only affect future
tenants, as they are created.  In the real world, this is rarely a
problem, since schedule and policy management tasks are few and
far between.

Lastly, we should talk about the things that are never shared:

  - **Targets** - The designers can think of no situation in which
    multiple tenants need to perform backup / restore operations
    against a shared data system.
  - **Jobs** - Without sharing targets, there isn't much point in
    sharing jobs.


Roles and Rights
----------------

Some times, we may want to grant someone access to our SHIELD
tenant, but only on restricted terms, with a subset of "full"
access.

For example, we may want to have a NOC team be able to re-run
failing backup jobs, but not be allowed to change any
configuration, or see the credentials stored in target / storage
endpoints.  This is where rights and roles come into play.

A **Right** is a token that governs one single aspect of the
system.  The ability to view backup archives would be a right; so
would the ability to run an ad hoc backup job.

A **Role** combines multiple _rights_ into a single assignable
unit.  "NOC Operator" might be a role that has just enough rights
to perform ad hoc backup jobs.

Each user is assigned one or more roles, for each tenant that they
have access to.

Rights are defined by the SHIELD software itself, and metadata
about those rights is made available for configuration.

Roles are globally defined by the SHIELD site administrators.

There is a special _pseudo-role_, called **Administrator** that
grants the holder full access; all rights on all tenants.


Auditing
--------

With tenants and roles, SHIELD is equipped to generate audit logs,
odetailing who carried out what transactions, when, as well as who
was denied access to perform a given task.

The key details of an audit log record are:

  - Who
    - Username / API Key
    - Affected Tenant (or global)
    - Remote IP
  - What
    - The Task
    - Involved Resources (if any)
  - When
  - Why
    - Was the user acting as an **Administrator**?
    - If not, what role granted the privilege


Tenant Backup and Restore
-------------------------

An outcome of introducing multi-tenancy to SHIELD is that teams
will now have data _inside_ SHIELD that they may wish to protect
(i.e. backup and restore), and should be allowed to do so without
affecting neighboring tenants.

This is distinctly different from SHIELD operators performing
full-SHIELD backup and restore operations to guard against
disaster scenarios.

To this end, we will need to write a custom target plugin for
SHIELD tenants that will:

  - Save just their encryption keys
  - Save all tenant-owned data:
    - Schedules
    - Retention Policies
    - Target Configurations
    - Jobs
    - Tasks
    - Archives
    - (Custom) Storage Endpoints
  - Save references to any _globally shared_ storage
    configuration.
    - UUID
    - Name
    - Plugin

If a tenant is not using any shared storage, the backup archives
can always be restored (assuming a functional SHIELD).

If a tenant _is_ using shared storage, restore operations will
refuse to execute if any shared references are not found in the
SHIELD.  The algorithm for checking the existence of these
resources works like this:

  1. Check if a globally-shared storage endpoint exists with the
     given UUID;  If so, nothing else needs to be updated.
  2. Check if a globally-shared storage endpoint exists with the
     same name, and plugin; if so, update all references to the
     old UUID (as stored in the backup archive) to the new UUID.
  3. Otherwise, halt the restore process (after checking
     suitability of other foreign references).


[enc]: https://github.com/starkandwayne/shield/blob/master/docs/encryption.md
