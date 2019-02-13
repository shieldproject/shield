log - Flexible Logging for Go
=============================

![Travis CI](https://travis-ci.org/jhunt/go-log.svg?branch=master)

Setup logging with `SetupLogging()`:

  - **Type**: logging mode to use - file, syslog, console
  - **Level**: debug, info, error, etc. (See all levels below.)
  - **Facility**: syslog facility to log to - daemon, misc, etc.
  - **File**: path to log to file if in file mode.

e.g.:

```go
log.SetupLogging(log.LogConfig{
    Type:  "console",
    Level: "warning"
})
```

If logging is not setup, then the messages will simply go to
`stdout`. If logging cannot be setup for `file` or `syslog`, then
the default `stdout` will be used. An error message will print to
`stderr` to notify you if this occurs.

The following log levels are defined, in decreasing verbosity /
increasing criticality:

  -  Debug
  -  Info
  -  Notice
  -  Warn
  -  Error
  -  Crit
  -  Alert
  -  Emerg

Usage
-----

Usage is the same as `Sprintf`/`Printf` statements - simply append
an `f` to the desired level. e.g.:

```go
dbug_mesg := "This isn't a bug."
log.Debugf("I really need to know this in debug mode: %s", msg)
```

Contributing
------------

1. Fork the repo
2. Write your code in a feature branch
3. Create a new Pull Request
