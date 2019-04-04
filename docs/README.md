SHIELD Documentation
====================

This directory contains the documentation for SHIELD.
Documents are grouped by audience: one for developers
(`docs/devs`) and another for operators (`docs/ops`).

Style Guide
-----------

Maintaining a consistent tone and voice throughout all of the
documentation is vital.  To that end, we adhere to the following
style guide:

  1. SHIELD is always uppercase, with no extra dots.
  2. Use informal first, or second person.

     For example, if explaining how to plan out where SHIELD and
     its agents get deployed, you might say the following:

     > Figuring out where the agents live is a matter of personal
     > opinion, driven by **your** situation.  **You will want
     > to** run the SHIELD agent as close as possible to the data
     > system as ...

  3. Less is more.  Shorter sentences have more power and impact
     than longer sentences.

  4. Never leverage a grandiloquent and ornate word when a
     diminutive one will suffice.

     Small words are okay.

  5. Use `back ticks`, in-line, for Commands, file names, and other
     computer-ish parts.

  6. Always spell-check.  Yes, it's a pain thanks to all the
     jargon and initialism, but it's worth it every time.

  7.  Use emphasis _sparingly_, but **use it when you need to.**

  8. Avoid the passive voice.


Building the Documentation
--------------------------

In the end, we want to maintain versioned copies of the rendered
documentation for each minor version (`x.y`) of SHIELD, on the
[shieldproject.io][1] website.

The `mkdocs` utility helps with that.  It does a few
things:

  - Create the directory structure
  - Resolves variables in the markdown to version-specific URLs
  - Renders HTML from the markdown

From the root of the codebase, run `./bin/mkdocs` to get the
help and usage information.


Screenshots / Terminal Output
-----------------------------

To help with consistency, we provide a Docker Compose recipe for
running a configured v8.x SHIELD core + agent.  Use this
environment any time you need a screenshot, or some terminal
output.

To use it (assuming you have Docker and Docker Compose
[installed][2]):

    $ cd docs
    $ docker-compose up

SHIELD should be available at http://localhost:9009/; you can
visit that in your browser, or hit it via the CLI.  The default
admin (failsafe) credentials are `admin` / `password`.


The Document Landscape
----------------------

- **Getting Started**

  Audience: operators who want to install SHIELD.

  Having read this, and following the (detailed) instructions in
  this document, an operator should have a running SHIELD core, a
  few agents, and have properly backed up and restored a single
  thing.  Their next step is the _Operations Manual_.

- **Quickstart**

  Audience: operators who just want to get SHIELD up and running,
  without reading too much; the `tl;dr` crowd.

  This is (more or less) the same document as _Getting Started_,
  in fewer words.

- **Operations Manual**

  Audience: operators who want to learn how to use SHIELD, in full
  detail.  This document also serves as reference material for more
  seasoned operators.

  The Operations Manual is _big_.  It covers all functional parts
  of SHIELD, including:

    - Architecture
    - The Web UI and CLI
    - Configuration
    - RBAC and Multi-tenancy
    - Plugins
    - Encryption

- **API Reference**

  Audience: developers who wish to understand more about how
  SHIELD works, either to integrate something _with_ SHIELD, or to
   fix / patch / extend the SHIELD software itself.

  (This one has some bespoke tooling that makes its compilation
   easier and less error-prone.)

- **Plugin Architecture**

  Audience: developers wishing to write / modify SHIELD plugins


[1]: https://shieldproject.io
[2]: https://docs.docker.com/compose/install/
