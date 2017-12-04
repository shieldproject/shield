SHIELD v8 is a marked improvement over previous version of SHIELD.

# New Features

  - *Multi-Tenancy* - SHIELD now supports the notion of tenants, which allow
    site operators to group their users logically, and sequester teams from
    one another.  Each tenant has its own set of jobs, tasks, archives,
    etc., and members of one tenant cannot interact with the resources of
    another.  Users can be assigned to multiple tenants, concurrently.

  - *Archive Encryption* - SHIELD now leverages AES-256 encryption when
    storing backup archives in cloud storage, making sure that your data is
    secure, even at-rest.

  - *Agent Registration* - SHIELD Agents now register with the SHIELD Core,
    and provide metadata to assist operators in the configuration of backup
    targets, and cloud storage systems.

  - *Improved Web UI* - SHIELD's web-based user interface got a massive
    overhaul in this release, with a concerted focus on efficiency and
    ease-of-use for operators, and their immediate concerns.

  - *New CLI* - The SHIELD CLI has been rewritten from the ground-up to
    interface more cleanly with the SHIELD v8 API.  It handles plugin
    configuration more naturally, without forcing you to write proper JSON.
    Yay.  It also supports a new `import` function that makes it easy to
    ensure that your target and storage systems, jobs, retention policies,
    etc. are always correct.

  - *Improved Scheduling* - Backup Jobs can now be run every X hours, much
    to the delight of SHIELD users everywhere.
