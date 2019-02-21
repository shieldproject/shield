The SHIELD Web UI - Developer Notes
===================================

This document contains notes for the initiate SHIELD Web UI
Developer, that curious creature who has decided to forego the
trappings of _modern Javascript development_ and live closer to
the land.  Or something.


A Note on Namespacing
---------------------

Not polluting the global namespace is all the rage these days.  We
took it a step too far, and put all of the definitions that we
need in a single namespace, `S.H.I.E.L.D`; yes -- that's a nested
structure 4 or 5 levels deep.  You're welcome.


The Data Structures At Play
---------------------------

The core engine of the SHIELD Web UI is the `Database` object.  Or
maybe we're calling it the `Engine` class, or just plain old `UI`.
Who can even keep up with the furious change of pace here??

Anyway, it's the only class we have, and it's pretty central to
the UI.  It governs how the templates are drawn, how data gets
pulled down via WebSocket and XMLHttpRequests, aggregates and
composites data elements, and more.

To keep things straight, this magical class maintains local state,
via variables that are publicly accessible, and made available
explicitly to template code.

To keep all of this straight, we present the Utmostly Correct and
One True Way of Accessing Data.

- `.shield` - Metadata about the SHIELD Core, including things set
  by the operator, like the environment name, color, motd.  Also
  includes version and IP information.

  Note: this element is always set, and it rarely changes.

- `.user` - A User object.  If the viewer is not authenticated,
  this will be _undefined_ (but present), and the reset of the
  elements are probably missing.

- `.tenant` - The UUID of the currently targeted tenant.  If the
  authenticated user has zero grants, this will be _undefined_.

- `.tenants` - A map of UUID &rarr; Tenant object, for all tenants
  the authenticated user belongs to.   Each Tenant object
  contains:

  - `.tenants[uuid].targets` - The target data systems defined for
    this tenant.

  - `.tenants[uuid].stores` - The storage systems defined for this
    tenant.  This _DOES NOT_ include global storage systems.

  - `.tenants[uuid].jobs` - Scheduled backup jobs for this tenant.
  - `.tenants[uuid].archives` - Backup archives for this tenant.

  - `.tenants[uuid].role` - The name of the role that this user
    has been granted, on this tenant (a string).

  - `.tenants[uuid].grants[right]` - A set of booleans, one for
    each tenant right, that translates roles like "admin" into
    answers to questions like "is this user an engineer on this
    tenant?"

    Currently recognized rights are: `admin`, `engineer`, and
    `operator`.

- `.system.grants[right]` - A set of booleans, one for each system
  right, that translate roles like "engineer" into answers to
  questions like "is this user an admin?"

  Currently recognized rights are: `admin`, `manager`, and
  `engineer`.

- `.global.stores` - A map of UUID &rarr; Store object, for all
   globally defined stores.  Admin accounts will have access to
   more (sensitive) attributes on each storage system.

- `.global.tenants` - A map of UUID &rarr; Tenant object, for all
  tenants.  This is only available for admins.

Some of this data is duplicated.  That is unfortunate, but the
structure described above serves the purposes of the web UI,
without burdening the frontend developer with too many performance
optimization shenanigans.

As data comes across the WebSocket, or via XMLHttpRequests, the
attributes will be merged in an overriding fashion.  Additional
fields not specified in an update are strictly to be left alone.
This allows the web UI to store additional metadata, or cache
calculated attributes, without fear of losing them later because
of a refresh or update.
