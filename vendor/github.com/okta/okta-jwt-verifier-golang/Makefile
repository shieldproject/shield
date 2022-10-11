GOFMT:=gofumpt
STATICCHECK=staticcheck
TEST?=$$(go list ./... |grep -v 'vendor')

default: build

ifdef TEST_FILTER
	TEST_FILTER := -run $(TEST_FILTER)
endif

build: fmtcheck
	go install

clean:
	go clean -cache -testcache ./...

clean-all:
	go clean -cache -testcache -modcache ./...

dep:
	go mod tidy

fmt: tools
	@$(GOFMT) -l -w .

fmtcheck:
	@gofumpt -d -l .

test:
	echo $(TEST) | \
		xargs -t -n4 go test -test.v $(TESTARGS) $(TEST_FILTER) -timeout=30s -parallel=4

tools:
	@which $(GOFMT) || go install mvdan.cc/gofumpt@v0.2.1
	@which $(STATICCHECK) || go install honnef.co/go/tools/cmd/staticcheck@2021.1.2

tools-update:
	@go install mvdan.cc/gofumpt@v0.2.1
	@go install honnef.co/go/tools/cmd/staticcheck@2021.1.2

vet:
	@go vet ./...
	@staticcheck ./...