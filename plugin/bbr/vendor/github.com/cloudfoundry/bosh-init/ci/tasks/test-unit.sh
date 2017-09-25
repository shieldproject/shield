#!/usr/bin/env bash

set -e -x

export PATH=/usr/local/ruby/bin:/usr/local/go/bin:$PATH
export GOPATH=$(pwd)/gopath

source /root/.bashrc

cd gopath/src/github.com/cloudfoundry/bosh-init
bin/require-ci-golang-version
bin/clean
bin/install-ginkgo
bin/test-prepare
bin/test-unit
