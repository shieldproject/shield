all: get-deps build

build:
	@go build pgstore.go

get-deps:
	@go get -d -v ./...

test: get-deps
	@go test -v ./...

format:
	@go fmt ./...
