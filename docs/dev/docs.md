Writing SHIELD Documentation
============================

This super-meta document explains how to write and review the
documentation in the SHIELD codebase.

Documentation is stored in the `docs/` director.  There are two
main sub-categories: Operator documentation in `docs/ops` and
Developer documentation in `docs/dev`.

Every document / guide / write-up exists as a markdown file in
it's sub-category directory, and a directory to contain images for
that document.  For example, assume we want to write a new
operator document, called _How To Install SHIELD on a Raspberry
Pi_.  We might start with the following:

    $ mkdir docs/ops/install-on-raspberry-pi
    $ vim   docs/ops/install-on-raspberry-pi.md

For the markdown document, write it as though it is the only set
of headers; start with an <h1>, and follow outline format from
there.

The Documentum Instance
-----------------------

To ensure that screenshots of SHIELD are uniform, we supply a
docker-compose recipe that spins up a fixed configuration of
SHIELD.  It can be found in `docs/docker-compose.yml`, and you run
it like this:

    $ docker-compose -f docs/docker-compose up

Then, you can access the SHIELD instance at http://localhost:9009.
You can log into it with the username `admin`, and the password
`password`, and may want to import the documentation data set:

    $ ./shield api http://localhost:9009 documentum
    $ ./shield -c documentum login
    $ ./shield -c documentum import docs/import.yml

Hyperlink References
--------------------

Because we intend to generate copies of the documentation, and
store them on a per-version basis, we need the documentation to be
relocatable.  Whenever you make references to other assets
(images, PDFs, other parts of the documentation), use the special
path `$docs/` as a stand-in for the _root of the `docs/`
directory_.  When referencing the SHIELD codebase, the special
path `$src/` can be used.  For example, to link to the
`core/main.go` file, use the path `$src/core/main.go`.

Generating the Documentation
----------------------------

To review the documentation for formatting and readability, you
will need to run the generation process.  This process creates an
embeddedable, relocated copy of the documentation in a new output
directory, while properly resolving the `$docs/` and `$src/`
placeholders to their appropriate values.

The process is all wrapped up in the `bin/mkdocs` script.  You can
run it like this:

    $ ./bin/mkdocs --version v8.1.x \
                   --output  tmp/docs \
                   --docroot file://$PWD/tmp/docs \
                   --style   basic

This should copy a bunch of files into a new `tmp/docs` directory
in the current working directory.  To visit the site in your
browser:

    $ open file://$PWD/tmp/docs

The _basic_ style provides enough boilerplate style and javascript
code to make the docs readable.  Namely, it provides a dynamically
generated sidebar for navigation, including all <h2> and <h3>
headings.
