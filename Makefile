# Run me to verify that all tests pass and all binaries are buildable before pushing!
# If you do not, then Travis will be sad.

# Everything; this is the default behavior
all-the-things: tests shield plugins

# Running Tests
tests: test
test:
	ginkgo *
	go vet ./...

# Running Tests for race conditions
race:
	ginkgo -race *

# Building Shield
shield:
	go build ./cmd/shieldd
	go build ./cmd/shield-agent
	go build ./cmd/shield-schema

# Building Plugins
plugin: plugins
plugins:
	go build ./plugin/dummy
	go build ./plugin/elasticsearch
	go build ./plugin/postgres
	go build ./plugin/redis
	go build ./plugin/s3

# Run tests with coverage tracking, writing output to coverage/
coverage: agent.cov db.cov plugin.cov supervisor.cov timespec.cov
%.cov:
	@mkdir -p coverage
	@go test -coverprofile coverage/$@ ./$*

report:
	go tool cover -html=coverage/$(FOR).cov

fixmes: fixme
fixme:
	@grep -rn FIXME * | grep -v Godeps/ | grep --color FIXME

# Deferred: Naming plugins individually, e.g. make plugin dummy
# Deferred: Looping through plugins instead of listing them
