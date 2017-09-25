<img src="https://cdn.rawgit.com/pivotal-cf/paraphernalia/master/docs/images/collection.svg" align="left" width="192px" height="192px"/>

# paraphernalia

> *useful go things*

[![Build Status](https://img.shields.io/travis/pivotal-cf/paraphernalia/master.svg?style=flat-square)](https://travis-ci.org/pivotal-cf/paraphernalia) [![Go Report Card](https://goreportcard.com/badge/github.com/pivotal-cf/paraphernalia?style=flat-square)](https://goreportcard.com/report/github.com/pivotal-cf/paraphernalia) [![Apache 2.0 License](https://img.shields.io/badge/license-Apache%202.0-blue.svg?style=flat-square)](/LICENSE) [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/pivotal-cf/paraphernalia)

<br />

## about

This is my flex-time project to build a higher-level internal standard library
for Go development inside Pivotal. I think that we do not have enough of these
projects so I'm putting my code where my mouth is.

The goal is to eventually reduce the boilerplate to create a new service until
you're only doing fun work. This is done by first building up a solid
collection of primitives which can then be composed together into a `Service`
abstraction which will handle things for you. If the opinions there don't work
for you then you can always drop down to the primitives below.

### why the name?

So I can troll you like I [trolled myself][fml].

[fml]: https://github.com/pivotal-cf/paraphernalia/commit/f1663e167ae262b81ef4f3cc28d951accb7890be

## usage

Please do! There are no backwards compatibility guarantees until things are a
littte further along. However, I'm not out to mess with you: things will stay
as similar as possible.

## image credit

"Collection" by [Karthik Srinivas](https://thenounproject.com/aathis/)
