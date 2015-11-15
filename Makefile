# Run me to verify that all tests pass and all binaries are buildable before pushing!
# If you do not, then Travis will be sad.

# Everything; this is the default behavior
all-the-things: tests shield plugins

# Running Tests
test:
	ginkgo *
tests: test

# Running Tests for race conditions
race:
	ginkgo -race *
# Building Shield
shield:
	go build ./cmd/shieldd
	go build ./cmd/shield-agent
	go build ./cmd/shield-schema

# Building Plugins
plugins:
	go build ./plugin/dummy
	go build ./plugin/elasticsearch
	go build ./plugin/postgres
	go build ./plugin/redis
	go build ./plugin/s3

# Deferred: Naming plugins individually, e.g. make plugin dummy
# Deferred: Looping through plugins instead of listing them
plugin: plugins
