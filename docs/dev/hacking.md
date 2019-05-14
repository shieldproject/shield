SHIELD Development
==================

This document contains notes on SHIELD, Golang, and life in
general, that are only of interest to people writing SHIELD code
itself.

Golang Dependencies
-------------------

We have ditched `godep` in favor of `govendor`.  You can install
`govendor` like this:

    go get github.com/kardianos/govendor
    go install github.com/kardianos/govendor

To **remove a dependency** that we no longer need:

    govendor remove github.com/markbates/goth/gothic

To **add a new dependency** manually:

    govendor add github.com/jhunt/go-cli

To **add all new dependencies, automagically**:

    govendor add +external

To **prune the vendored dependencies**:

    govendor remove +unused

To **see what is currently unnused**:

    govendor list +unused
