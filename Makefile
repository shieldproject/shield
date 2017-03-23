# Run me to verify that all tests pass and all binaries are buildable before pushing!
# If you do not, then Travis will be sad.

export GO15VENDOREXPERIMENT=1

BUILD_TYPE?=build

# Everything; this is the default behavior
all: format tests shield plugins

# go fmt ftw
format:
	go list ./... | grep -v vendor | xargs go fmt

# Running Tests
tests: test
test:
	go test ./...
	./t/api

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
	go $(BUILD_TYPE) ./plugin/xtrabackup
	go $(BUILD_TYPE) ./plugin/rabbitmq-broker
	go $(BUILD_TYPE) ./plugin/scality
	go $(BUILD_TYPE) ./plugin/consul
	go $(BUILD_TYPE) ./plugin/mongo


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

ARTIFACTS := artifacts/shield-server-{{.OS}}-{{.Arch}}
LDFLAGS := -X main.Version=$(VERSION)
release:
	@echo "Checking that VERSION was defined in the calling environment"
	@test -n "$(VERSION)"
	@echo "OK.  VERSION=$(VERSION)"

	gox -osarch="linux/amd64" -ldflags="$(LDFLAGS)" --output="$(ARTIFACTS)/plugins/fs"                ./plugin/fs
	gox -osarch="linux/amd64" -ldflags="$(LDFLAGS)" --output="$(ARTIFACTS)/plugins/docker-postgres"   ./plugin/docker-postgres
	CGO_ENABLED=1 gox -osarch="linux/amd64" -ldflags="$(LDFLAGS)" --output="$(ARTIFACTS)/plugins/dummy"             ./plugin/dummy
	CGO_ENABLED=1 gox -osarch="linux/amd64" -ldflags="$(LDFLAGS)" --output="$(ARTIFACTS)/plugins/elasticsearch"     ./plugin/elasticsearch
	CGO_ENABLED=1 gox -osarch="linux/amd64" -ldflags="$(LDFLAGS)" --output="$(ARTIFACTS)/plugins/postgres"          ./plugin/postgres
	CGO_ENABLED=1 gox -osarch="linux/amd64" -ldflags="$(LDFLAGS)" --output="$(ARTIFACTS)/plugins/redis-broker"      ./plugin/redis-broker
	CGO_ENABLED=1 gox -osarch="linux/amd64" -ldflags="$(LDFLAGS)" --output="$(ARTIFACTS)/plugins/s3"                ./plugin/s3
	CGO_ENABLED=1 gox -osarch="linux/amd64" -ldflags="$(LDFLAGS)" --output="$(ARTIFACTS)/plugins/mysql"             ./plugin/mysql
	CGO_ENABLED=1 gox -osarch="linux/amd64" -ldflags="$(LDFLAGS)" --output="$(ARTIFACTS)/plugins/rabbitmq-broker"   ./plugin/rabbitmq-broker
	CGO_ENABLED=1 gox -osarch="linux/amd64" -ldflags="$(LDFLAGS)" --output="$(ARTIFACTS)/plugins/scality"           ./plugin/scality
	CGO_ENABLED=1 gox -osarch="linux/amd64" -ldflags="$(LDFLAGS)" --output="$(ARTIFACTS)/plugins/mongo"             ./plugin/mongo
	CGO_ENABLED=1 gox -osarch="linux/amd64" -ldflags="$(LDFLAGS)" --output="$(ARTIFACTS)/plugins/consul"            ./plugin/consul
	CGO_ENABLED=1 gox -osarch="linux/amd64" -ldflags="$(LDFLAGS)" --output="$(ARTIFACTS)/plugins/xtrabackup"        ./plugin/xtrabackup

	CGO_ENABLED=1 gox -osarch="linux/amd64" -ldflags="$(LDFLAGS)" --output="$(ARTIFACTS)/agent/shield-agent"        ./cmd/shield-agent

	CGO_ENABLED=1 gox -osarch="linux/amd64" -ldflags="$(LDFLAGS)" --output="$(ARTIFACTS)/cli/shield"                ./cmd/shield

	CGO_ENABLED=1 gox -osarch="linux/amd64" -ldflags="$(LDFLAGS)" --output="$(ARTIFACTS)/daemon/shield-schema" ./cmd/shield-schema
	CGO_ENABLED=1 gox -osarch="linux/amd64" -ldflags="$(LDFLAGS)" --output="$(ARTIFACTS)/daemon/shieldd"       ./cmd/shieldd

	rm -f artifacts/*.tar.gz
	cd artifacts && for x in shield-server-*; do cp -a ../webui/ $$x/webui; cp ../bin/shield-pipe $$x/daemon; tar -czvf $$x.tar.gz $$x; rm -r $$x;  done

.PHONY: shield
