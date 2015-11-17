# Run me to verify that all tests pass and all binaries are buildable before pushing!
# If you do not, then Travis will be sad.

BUILD_TYPE?=build

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
	go $(BUILD_TYPE) ./cmd/shieldd
	go $(BUILD_TYPE) ./cmd/shield-agent
	go $(BUILD_TYPE) ./cmd/shield-schema

# Building Plugins
plugin: plugins
plugins:
	go $(BUILD_TYPE) ./plugin/dummy
	go $(BUILD_TYPE) ./plugin/elasticsearch
	go $(BUILD_TYPE) ./plugin/postgres
	go $(BUILD_TYPE) ./plugin/redis
	go $(BUILD_TYPE) ./plugin/s3

# Run tests with coverage tracking, writing output to coverage/
coverage: agent.cov db.cov plugin.cov supervisor.cov timespec.cov
%.cov:
	@mkdir -p coverage
	@go test -coverprofile coverage/$@ ./$*

report:
	go tool cover -html=coverage/$(FOR).cov

fixmes: fixme
fixme:
	@grep -rn FIXME * | grep -v Godeps/ | grep -v README.md | grep --color FIXME || echo "No FIXMES!  YAY!"

# Deferred: Naming plugins individually, e.g. make plugin dummy
# Deferred: Looping through plugins instead of listing them
