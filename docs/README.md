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

  7. Use emphasis _sparingly_, but **use it when you need to.**

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

To generate _working copy_ documentation, suitable for hosting
with something like [gow](https://github.com/jhunt/gow):

    $ ./bin/mkdocs --version 8.2.0   \
                   --docroot /docs   \
                   --output tmp/docs \
                   --style basic
    [mkdir]  tmp/docs/ops
    [render] tmp/docs/ops/architecture.md
    [render] tmp/docs/ops/getting-started.md
    [render] tmp/docs/ops/plugins.md
    [render] tmp/docs/ops/manual.md
    [mkdir]  tmp/docs/ops/architecture
    [copy]   tmp/docs/ops/architecture/agent.png
    [copy]   tmp/docs/ops/architecture/webui.png
    [copy]   tmp/docs/ops/architecture/database.png
    [copy]   tmp/docs/ops/architecture/overview.png
    ... etc ...

    $ (cd tmp/doc && gow)
    binding *:3001 to serve / -> .

You can now access the documentation at
<http://127.0.0.1:3001/docs>.

You can also just run

    make docs

:)


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

There is a SHIELD import file, `docs/documentum.yml`, that you can
then use to re-inflate all of the test data.  This import data set
will evolve and grow over time, so add to it as needed.

    $ cd docs
    $ shield api http://localhost:9009 documentum
    $ shield -c documentum login
      ... provide admin creds ...
    $ shield -c documentum import documentum.yml

Now you have a fully-populated SHIELD instance that will stay
consistent across developers, machines, and environments!  Go you!

Firefox has this wonderful thing called _Responsive Mode_.  It's
actually for testing out mobile platforms without needing one of
every phone / tablet / phablet that has ever been made, but it has
a few interesting features that make it ideal for distributed
screenshot taking:

  1. It allows you to set the screen dimensions
  2. It has a _screenshot_ button

You can access it via the context menu > Web Developer >
Responsive Design Mode.  If you prefer shortcuts, its ⌥⌘ (at least
on macOS).

All screenshots are 1200px wide, unless you have a very good
reason to need something smaller.  Scrensshots should be as tall
as necessary to show what needs showing, but no taller.


Asset Organization
------------------

Each document exists as a `.md` file (containing the markdown),
and a directory, that share a short name.  For example, the
_Getting Started_ document's short name is `getting-started`.  If
you look in the `docs/ops` folder, you will see the following:

- `getting-started.md` - The text of the document.
- `getting-started/` - All assets pertinent to the document.

This helps us avoid naming collisions that we would otherwise run
into with a shared directory for images, SVG files, YAML, etc.

When the documentation is built into its final HTML form, the
markdown file will be saved _into_ the asset folder as
`index.html`.  You'll want to avoid naming an asset `index.html`
for this reason.  This also lets you link to or embed other assets
using relative HTTP paths.  For example:

    <img src="intro.png">

Will resolve from inside of the `getting-started/` directory, and
find the appropriate image.  This is the most portable way, that
allows us (and our pipelines!) to auto-generate and relocate
instances of the documentation, as need arises.


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

- **SHIELD Architecture**

  Audience: operators evaluating SHIELD to determine if it is a
  good fit for their use-case, from the architectural "pieces and
  parts" standpoint.

  Required reading for all developers wishing to get involved in
  contributing to SHIELD, and for all operators who want to truly
  understand the guts of SHIELD.

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
