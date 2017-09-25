#!/usr/bin/env bash

set -ex

export AWS_AMI_ID=$(cat windows-ami/version)
export PATH="/usr/local/ruby/bin:/usr/local/go/bin:$PATH"
export GOPATH="$(pwd)/gopath"

cd gopath/src/github.com/cloudfoundry/bosh-agent
bin/test-integration-windows
