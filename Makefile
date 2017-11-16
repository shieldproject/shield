# Run me to verify that all tests pass and all binaries are buildable before pushing!
# If you do not, then Travis will be sad.

BUILD_TYPE?=build

# Everything; this is the default behavior
all: format shieldd buckler shield-agent shield-schema plugins test

# go fmt ftw
format:
	go list ./... | grep -v vendor | xargs go fmt

# Running Tests
test: go-tests api-tests plugin-tests
plugin-tests: plugins
	go build ./plugin/mock
	./t/plugins
	@rm -f mock
go-tests:
	go list ./... | grep -v vendor | xargs go test
api-tests:
	./t/api

# Running Tests for race conditions
race:
	ginkgo -race *

# Building Shield
shield: shieldd shield-agent shield-schema buckler

shieldd:
	go $(BUILD_TYPE) ./cmd/shieldd
shield-agent:
	go $(BUILD_TYPE) ./cmd/shield-agent
shield-schema:
	go $(BUILD_TYPE) ./cmd/shield-schema

buckler: cmd/buckler/help.go
	go $(BUILD_TYPE) ./cmd/buckler
help.all: cmd/buckler/main.go
	grep case $< | grep '{''{{' | cut -d\" -f 2 | sort | xargs -n1 -I@ ./buckler @ -h > $@

# Building Plugins
plugin: plugins
plugins:
	go $(BUILD_TYPE) ./plugin/dummy
	for plugin in $$(cat plugins); do \
		go $(BUILD_TYPE) ./plugin/$$plugin; \
	done


demo: clean shield plugins
	./demo/build
	(cd demo && docker-compose up)

docs: docs/API.md
docs/API.md: docs/API.yml
	perl ./docs/regen.pl <$+ >$@

clean:
	rm -f shieldd shield-agent shield-schema shield
	rm -f $$(cat plugins) dummy


# Assemble the CLI help with some assistance from our friend, Perl
HELP := $(shell ls -1 cmd/buckler/help/*)
cmd/buckler/help.go: $(HELP) cmd/buckler/help.pl
	./cmd/buckler/help.pl $(HELP) > $@


# Run tests with coverage tracking, writing output to coverage/
coverage: agent.cov db.cov plugin.cov supervisor.cov timespec.cov
%.cov:
	@mkdir -p coverage
	@go test -coverprofile coverage/$@ ./$*

report:
	go tool cover -html=coverage/$(FOR).cov

fixmes: fixme
fixme:
	@grep -rn FIXME * | grep -v vendor/ | grep -v README.md | grep --color FIXME || echo "No FIXMES!  YAY!"

dev:
	./bin/testdev

# Deferred: Naming plugins individually, e.g. make plugin dummy

init:
	go get github.com/kardianos/govendor
	go install github.com/kardianos/govendor

save-deps:
	govendor add +external

ARTIFACTS := artifacts/shield-server-linux-amd64
LDFLAGS := -X main.Version=$(VERSION)
release:
	@echo "Checking that VERSION was defined in the calling environment"
	@test -n "$(VERSION)"
	@echo "OK.  VERSION=$(VERSION)"
	export GOOS=linux GOARCH=amd64
	for plugin in $$(cat plugins); do \
	              go build -ldflags="$(LDFLAGS)" -o "$(ARTIFACTS)/plugins/$$plugin"     ./plugin/$$plugin; \
	done
	              go build -ldflags="$(LDFLAGS)" -o "$(ARTIFACTS)/agent/shield-agent"   ./cmd/shield-agent
	              go build -ldflags="$(LDFLAGS)" -o "$(ARTIFACTS)/cli/shield"           ./cmd/buckler
	CGO_ENABLED=1 go build -ldflags="$(LDFLAGS)" -o "$(ARTIFACTS)/daemon/shield-schema" ./cmd/shield-schema
	CGO_ENABLED=1 go build -ldflags="$(LDFLAGS)" -o "$(ARTIFACTS)/daemon/shieldd"       ./cmd/shieldd

	rm -f artifacts/*.tar.gz
	cd artifacts && for x in shield-server-*; do cp -a ../web2/htdocs $$x/webui; cp ../bin/shield-pipe $$x/daemon; tar -czvf $$x.tar.gz $$x; rm -r $$x;  done


JAVASCRIPTS := web2/src/js/jquery.js
JAVASCRIPTS += web2/src/js/lib.js
JAVASCRIPTS += web2/src/js/sticky-nav.js
JAVASCRIPTS += web2/src/js/shield.js
web2/htdocs/shield.js: $(JAVASCRIPTS)
	cat $+ >$@

web2: web2/htdocs/shield.js

.PHONY: shield plugins dev web2 buckler shieldd shield-schema shield-agent demo
