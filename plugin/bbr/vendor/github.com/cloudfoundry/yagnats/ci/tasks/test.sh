#!/usr/bin/env bash
set -e

export GOPATH=$(pwd)/gopath
export PATH=$(pwd)/gopath/bin:$PATH

cd gopath/src/github.com/cloudfoundry/yagnats

go get -v github.com/nats-io/gnatsd
go get gopkg.in/check.v1
go get -v ./...
go build -v ./...

go test -race
