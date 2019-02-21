SHIELD Web UI Design Notes
==========================

SHIELD needs a better Web User Interface.  Here, we have collected
our thoughts on the matter, and supplied some design mocks.

This undertaking is ambitious.  It involves not just a visual
refresh of the existing functionality, but new features and
behaviors to provide a better user experience.

Main Dashboard
--------------

May I present the new SHIELD dashboard.

![Main Dashboard Image](images/uiv2.1b.png)

It sure is pretty.

### 1. Top Bar

The **Top Bar** provides a minimal global identification space on the left-hand side, and current user indication on the right-hand side.

On the left, the word "SHIELD" is always followed by the name of this SHIELD (which is managed by the system implementor / BOSH deployment team), and then the non-loopback IP VM IP address.  This allows an operator accessing the web UI to unequivocally identify _which_ SHIELD they are working on.

On the right, the currently logged-in user's display name will be shown.  This feature doesn't make a whole lot of sense until we get to RBAC/MT -- Role-Based Access Control / Multi-Tenancy.  Administrators who have access to all things, including shared resources, will have the text `(admin)` appended after their display name, to remind them that they have elevated privileges.  The hamburger icon on the far right expands a user-centric menu for things like password changes, profile management, etc (not shown). 

### 2. Story Sidebar

The **Story Sidebar** is an attempt to provide operators with more procedural interfaces for managing SHIELD.  We have observed several people of varying skill levels attempt to navigate the previous web user interface; they primarily have solid, concrete objectives for each session with the UI.  If we can capture those stories, and encode them into web forms and other user interfaces, SHIELD will be much easier to use.

Each story has a name, like _Perform an ad hoc backup_, and a description.  These help operators find the story they wish to follow.  The Story Sidebar shows these two bits of information.

Ideally, we will need only a small number of these stories.  If they can all comfortably fit on the sidebar, then the displayed search interface (_What shall we do?_) can be omited.

### 3. Heads-Up Display

The **Heads-Up Display** serves to provide operators with tactical / operational feedback on the current state and health of this SHIELD instance.  There is a fine line to walk here -- not enough data, and the HUD is useless; too much, and we risk becoming our own monitoring system.

The HUD is split into three segments: _Instance Information_, _Health Indicators_, and _Data Protection Summary_.  These sections are described below.

#### Instance Information

Here, we provide more information about the SHIELD instance itself, including:

- **Version** - The version of the SHIELD daemon
- **Instance IP** - The non-loopback IP address of this SHIELD instance.  We may be able to handle multiple IP addresses here, whereas the Top Bar is limited to just one address.
- **Base URL** - The base URL of this instance, for accessing it by name.
- **Name** - The name of the SHIELD instance, as set by the deployment manifest.  The screen mock shows _PRODUCTION_ written in green; it would be a nice touch to enable the deployment manifest to provide a "color", for teams that which to color-code their deployments (perhaps red for prod, green for sandbox).

#### Health Indicators

We have identified the following key health indicators for the SHIELD instance's base functionality:

1. Whether the SHIELD API is responding
2. Whether Cloud Storage is accessible
3. Whether there are any failed or pending jobs

It may seem odd to have a check on the SHIELD API from the SHIELD UI, especially since they are, ostensibly, both served by the same daemon process.  However, the UI only _loads_ from the server; it executes entirely on the client.  It is quite possible for an operator to keep a SHIELD UI window open for hours, days, or even weeks.

The "Cloud Storage" health indicator is novel.  It reports on any problems related to the _storage_ of backup archives, by testing the viability of storage endpoint configurations.  This is new behavior, that requires changes to both the SHIELD daemon, and the plugin framework as well.

We can implement this in one of two ways.

The simplest way involves modifying the plugin framework to provide a new action, `test`, which will not only validate the configuration, but also perform some small viability test.  For example, the `s3` plugin may write a small file, and then read it back to verify that both store and retrieve operations work.  Then, the SHIELD daemon could be set to run these tests on a set schedule -- if any of them fail, the HUD would reflect the failure.

A more nuanced implementation would take into account the SHIELD agents being leveraged in performing backups.  For each storage endpoint, determine what SHIELD agents could be called upon to connect to it, and execute the tests from those agents.  This more closely matches reality, and would catch environmental issues like intervening firewalls or IaaS security groups.  It does, however, require changes to the SHIELD agents as well as the core and the plugin suite.

The final health indicator shows green if all scheduled jobs have run at least once, and the most recent run (ad hoc or scheduled) has succeeded.  This is similar to the current UI, which provides a list of jobs that are failing, in a _warning banner_.

For each of these health indicators, clicking on it when it is in the "failing" state should take the operator to a page detailing exactly what is wrong, with links to other relevant parts of the system.  For example, for failing jobs, the link would go to a page listing off exactly which jobs are failing, with further links to each individual job, to allow reconfiguration and retries.

#### Data Protection Summary

The **Data Protection Summary** provides tactical information that may or may not have a "health" component, but is nonetheless useful to operators.

- **Scheduled Backup Jobs** - How many backup jobs have been scheduled.  We believe that operators have a gut feeling for how large or small their SHIELD instances are, based on their environment (production vs. staging), their placement (us-east-1 vs. on-prem datacenter #2), etc.  Showing them this information lets them check reality against their expectation.
- **Backup Archives** - How many backup archives currently exist, across all storage backends.  This is a surprisingly difficult question to answer with today's version of SHIELD.
- **Cloud Storage Used** - How much space is being used by resident backup archives.  The primary cost in any SHIELD deployment is the storage pool.  Amazon's S3 offering is cheap, but definitely not free.  Operators ought to be able to see just how much space they are using, to aide them in understanding the ramifications of frequent scheduling and long retention policies.
- **Daily Storage Increase** - How much Cloud Storage Usage is expected to increase, on a daily basis, given current frequencies, retention policies, configured backup jobs, and historical data about backup archive sizes.

The first two metrics are easily obtained with the current data model.

The storage usage metrics will require changes to the API and the database schema, to track archive size and return summarizations.

The _Daily Storage Increase_ metric is interesting, from a calculation point-of-view; we shall see how well it fares in the real world.  To determine this value, SHIELD will need to perform a linear fit on time-series data regarding backup archive sizes, taking into account archive purgation due to retention policy.  Once each backup job has an linear approximation, it should be trivial to combine these and come up with a projection for daily increase (or decrease).

Showing a storage usage delta may give operators a rough idea of their scaling and capacity requirements.  This calculation also figures into some screen mocks shown later.

An underlying assumption of this model is that backup archives change size linearly.  This may not bear out in practice and if it does not, this feature will have to be scrapped.

#### 4. Content Area

The main content goes here.  What exactly that means is highly dependent on what you are doing in the SHIELD web UI.



Backup Jobs
-----------

The configured backup jobs are displayed in a card interface:

![Backup Jobs Card Interface](images/uiv2.1.png)

Each card provides at least the following information:

- The name of the backup job, i.e. _BOSH Database Backup_.
- The job's schedule, in a human readable format.
- The job's retention policy, expressed in days.
- A more human-friendly description of what the schedule + retention policy means -- how many archives will _actually_ be kept?
- A health indicator.
- A visual summary of the target ⇔ store configuration (with target on the left, store on the right, and a non-trivial line connecting the two)

When the mouse hovers over the card, edit / delete icons will appears, as can be seen in the _BOSH Database Weekly_ card.

The _CF UAA Database_ card shows what a failing job looks like; the health indicator is visibly bigger than (almost 1.75x) the size of the "ok" indicator.  The name of the job is also rendered in a scary red font.

Clicking anywhere on the card will bring up the Job page (discussed later).

So far, this screen has not introduced any new functionality, outside of perhaps a minor tweak to the API to calculate the number of kept archives.

However, we may want to modify the _{target, schedule, retention policy, store}_ tuple, and allow a job to be _{target, store}_, with one or more _{schedule, retention policy}_ elements.  The bottom-left card, _BOSH Database Backup_ shows this scheme in action.  Here, BOSH is being backed up weekly for 3 months, and daily for a week.  This scheme has a few advantages:

- It is trivial and uncomplicated to show all of the archives for a given target system, and let the operator choose the one they want, withcout forcing them to hop between mostly unrelated jobs.
- Operators do not have to practice the dark art of _name mangling_ by appending " - Daily" or " - Weekly" to their job names to make them unique and unambiguous.

The primary downside of this approach is, of course, migrating data.  Perhaps we should survey the community to see if anyone is availing themselves of the multi-schedule capabilities of SHIELD.

The detail page for a backup job looks like this:

![Backup Job Detail Page](images/uiv2.4.png)

The card from the listing page is repeated, in the top left.  To the right, any summary / notes provided by operators during job configuration should be displayed.  These are not expected to be terribly lengthy, so the side-by-side format should suffice.

For the most part, people access job details to perform one of two actions: (a) verification of scheduling / configuration and (b) to find an archive and restore it.

The next section assists with (b), by listing out all of the archives generated for this job.  This is where hanging _{schedule, retention policy}_ as a one-to-many off of _{target, store}_ comes in handy.  The table lists the schedule that each archive was taken under, when the backup was performed, what retention policy governs the archive, how big the archive is, and provides a link to view task details (perhaps in a lightbox).  The two icons on the far right enable manual download of archives, and the restoration of archives.

Some operators have requested the ability to annotate backup archives.  This is shown in the third archive.  This has several uses, not the least of which is accounting for ad hoc job runs (especally failing ones) with more detail and context.

The final section of this page, _Failed Task Runs_, will contains a list of non-ok tasks associated with this job, for operator review (and possible annotation).


Storage Endpoints
-----------------

As we said before, the primary _cost_ of SHIELD is its storage.  Having better tools to understand the impact SHIELD is having on storage, how it is using storage, etc. makes using and scaling SHIELD easier.

![Storage Endpoint Mock](images/uiv2.2.png)

The name of the storage endpoint, as assigned by the operator, is shown on the left, in large font.  The plugin is shown to the right, in the same blue, rounded-corner rectangle as the Backup Jobs Cards use.

The interesting bit, however, comes from the usage meter in the middle.

If we modify SHIELD to track the size of each archive, it becomes trivial to calculate the total size of all archives in a given storage endpoint.  This value can be plotted.

The _Threshold_ value in the middle is something we imagine conscientious operators will want to set, both on pay-as-you-go solutions like Amazon S3, and on local disk / SAN offerings.  What SHIELD does with this threshold is currently up in the air.  It seems a bit harsh to suspend backup activities until the usage is back under the threshold (presumably, because of either manual or automatic purges).  At the very least, we should make the data available to any interested monitoring systems.

The _Projected_ value is also of interest.  Given the same calculations that underly the daily storage increase metric in the HUD, we can project out storage usage on this endpoint by only considering jobs that involve this store.  If we put a time frame on the projection (6 months, 1 years, etc.), we can provide some (hopefully) meaningful data to backup system operators, arming them to either reduce retention periods, reduce schedule frequency, or increase backend storage, before it becomes a problem.

To accommodate this particular view, we will need to add some new fields to the `stores` table, for tracking threshold name / value, and a new configuration option to the SHIELD daemon for setting the projection time frame.

The _Storage Details_ section contains additional pertinent information.  Some of it may not be necessary.  Redundant information (like _Storage Used_) may be omited in the final UI.  Particularly, we would like to see the configuration details as understood by the plugin configuration.  In the screen mock, the S3 host, access key and prefix path are shown.  We probably need to modify the plugin framework to allow plugins to generate the metadata that goes here, in some presentation agnostic format (i.e. not HTML).


Agent Inspection
----------------

One thing missing from the current web UI is the ability to see what agents are registered with the system, what version of SHIELD they are running, and what capabilities they have.  The UI doesn't show this information because it cannot.  Both the SHIELD agent implementation and the SHIELD daemon core need to be modified to provide this level of reflection.  For several reasons, this change is on the roadmap.

![Agents of SHIELD](images/uiv2.3.png)

Each known agent is listed in the table, with its:

- **Name** - The name of the agent, as set by the deployment.  In BOSH-land, this will probably take the form of _deployment/instange/index_
- **Address** - The IP address (shown) and port (not shown) that the SHIELD core uses to contact the SHIELD agent.
- **Version** - What version of SHIELD is the agent running.  More on this in a bit.
- **Storage** - Initially, this was meant to show the amount of storage used by jobs run on this agent.  That may not be a terribly relevant number.
- **Status** - The health status of the agent.  Once we have a regular system of pinging the agent, we can perform remote health checks and get more detail about the ability of an agent to perform its duties.
- **Last Seen** - When the SHIELD core last saw health / registration information from this agent.  This seeks to inform operators who will probably abuse the next column...
- **Ping** - The satellite dish icon lets operators initiate an immediate ping to the agent to see if its state has changed.  This can come in quite handy, for example, after you have fixed a misconfigured firewall, or updated a security group.

SHIELD is a highly distributed system, more so than even Cloud Foundry (although it has far fewer moving parts!)  It can be difficult to keep all components of the overarching system on the same version, since different environments may be subject to different change windows, waiting periods, etc.  Once we have version information that can be queried for all parts of the system, we can use it to warn operators of known inconsistencies.

The screen mock shows just such an example, with a hypothetical incompatibility between SHIELD core 6.5.0+, and SHIELD agents prior to 6.3.0.  In this (fabricated) scenario, the cf-prod/uaadb/0 agent is failing to report in because of an upgrade.

Tracking these inconsistencies will not be easy.  However, we believe it will be worth it, by enabling operators to recover in the face of incompatibility, without tying the hands of the developers with respect to backwards compatibility.

Future Direction
================

This document is by no means complete.  There are several other screens that need to be mocked out, and other UI elements to be designed, considered and implemented.