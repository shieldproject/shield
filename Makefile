# Run me to verify that all tests pass and all binaries are buildable before pushing!
# If you do not, then Travis will be sad.

BUILD_TYPE?=build

# Everything; this is the default behavior
all: format tests shield plugins

# go fmt ftw
format:
	go fmt ./...

# Running Tests
tests: test
test:
	ginkgo * ./cmd/shield
	go vet ./...

# Running Tests for race conditions
race:
	ginkgo -race *

# Building Shield
shield:
	go $(BUILD_TYPE) ./cmd/shieldd
	go $(BUILD_TYPE) ./cmd/shield-agent
	go $(BUILD_TYPE) ./cmd/shield-schema
	go $(BUILD_TYPE) ./cmd/shield

# Building the Shield CLI *only*
shield-cli:
	go $(BUILD_TYPE) ./cmd/shield

# Building Plugins
plugin: plugins
plugins:
	go $(BUILD_TYPE) ./plugin/fs
	go $(BUILD_TYPE) ./plugin/docker-postgres
	go $(BUILD_TYPE) ./plugin/dummy
	go $(BUILD_TYPE) ./plugin/elasticsearch
	go $(BUILD_TYPE) ./plugin/postgres
	go $(BUILD_TYPE) ./plugin/redis-broker
	go $(BUILD_TYPE) ./plugin/s3
	go $(BUILD_TYPE) ./plugin/mysql
	go $(BUILD_TYPE) ./plugin/rabbitmq-broker

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

.PHONY: shield
