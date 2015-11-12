# Shield CLI Architecture Overview

## Generalized Architecture

The `shield` command provides the cli for interfacing with shield.  It is
composed of the front-end cli parser and results display and the backend api
agent.

```
               =============================== shield ==========================
               +-- cli package ----------------+   +--api_agent package -------+
               |                               |   |                           |
[terminal] <-> |[parser] <-> [validator/output]|<->|[packager] <-> [api caller]| <-> [api host]
               |                               |   |                           |
               +-------------------------------+   +---------------------------+

```

The root parser is in the `main.go` package and is responsible for providing
the verb commands such as list, show, edit, and delete for the component
subcommands to hook into.  This parser is based on github.com/spf13/cobra.

The components each have their own cli file to provide specific parsing for
their actions, located in files named *\<plural of component>*`_cli.go` (i.e.:
`targets_cli.go`)

This file also parses the specific options for the action, and passes them on
to the specific packager for that component.

The packager is cli-agnostic, and can be used by anything providing the
correct options to the packager.  It determines the correct url needed for the
request, calls the api caller, and returns a the struct or slice of structs
appropriate for the given component being accessed.  There is one packager for
each component type, and is named the same as the corresponding validator but
without the `_cli`

The api caller is a generalized http requester with support for json results,
which it marshals into the correct struct given by the packager.

For example:  `shield list targets` will be parsed by the main cobra parser,
passed to `processListTargetsRequest` which will validate and format options,
which it then passes to `api_agent.FetchListTargets`.  This is the packager
that determines the url for the request and assembles it with the passed
options, and makes the call to `api_agent.makeApiCall`

The api_agent package component should behave like a library and can be used
to create other client applications such as a web frontend or tool to generate
reports.  This can also be used by monitoring tools to query status.

## To add a component
***TBD***  This has been left as an exercise to the reader
