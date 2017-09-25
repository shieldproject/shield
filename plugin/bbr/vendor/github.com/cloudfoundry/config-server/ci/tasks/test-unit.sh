#!/bin/sh
set -e -x

export GOPATH=$(pwd)
export PATH=/usr/local/go/bin:$GOPATH/bin:$PATH

go clean -r github.com/cloudfoundry/config-server

go get github.com/onsi/ginkgo/ginkgo
go get github.com/onsi/gomega

cd src/github.com/cloudfoundry/config-server
ginkgo -r -trace -skipPackage="integration,vendor"