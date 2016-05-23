# Run me to verify that all tests pass and all binaries are buildable before pushing!
# If you do not, then Travis will be sad.

export GO15VENDOREXPERIMENT=1

BUILD_TYPE?=build
GO_CMD?=godep go

# Everything; this is the default behavior
all: format tests shield plugins

# go fmt ftw
format:
	go list ./... | grep -v vendor | xargs go fmt

# Running Tests
tests: test
test:
	ginkgo * ./cmd/shield
	go list ./... | grep -v vendor | xargs go vet

# Running Tests for race conditions
race:
	ginkgo -race *

# Building Shield
shield:
	$(GO_CMD) $(BUILD_TYPE) ./cmd/shieldd
	$(GO_CMD) $(BUILD_TYPE) ./cmd/shield-agent
	$(GO_CMD) $(BUILD_TYPE) ./cmd/shield-schema
	$(GO_CMD) $(BUILD_TYPE) ./cmd/shield

# Building the Shield CLI *only*
shield-cli:
	$(GO_CMD) $(BUILD_TYPE) ./cmd/shield

# Building Plugins
plugin: plugins
plugins:
	$(GO_CMD) $(BUILD_TYPE) ./plugin/fs
	$(GO_CMD) $(BUILD_TYPE) ./plugin/docker-postgres
	$(GO_CMD) $(BUILD_TYPE) ./plugin/dummy
	$(GO_CMD) $(BUILD_TYPE) ./plugin/elasticsearch
	$(GO_CMD) $(BUILD_TYPE) ./plugin/postgres
	$(GO_CMD) $(BUILD_TYPE) ./plugin/redis-broker
	$(GO_CMD) $(BUILD_TYPE) ./plugin/s3
	$(GO_CMD) $(BUILD_TYPE) ./plugin/mysql
	$(GO_CMD) $(BUILD_TYPE) ./plugin/rabbitmq-broker

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

dev: shield
	./bin/testdev

# Deferred: Naming plugins individually, e.g. make plugin dummy
# Deferred: Looping through plugins instead of listing them

restore-deps:
	godep restore ./...

save-deps:
	godep save ./...

.PHONY: shield
