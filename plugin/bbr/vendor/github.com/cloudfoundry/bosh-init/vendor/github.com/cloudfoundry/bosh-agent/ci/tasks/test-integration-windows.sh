#!/usr/bin/env bash

set -ex

export AGENT_ZIP_URL=$(cat bosh-agent-zip/url)
export AGENT_DEPS_ZIP_URL=$(cat bosh-agent-deps-zip/url)
export PATH=/usr/local/ruby/bin:/usr/local/go/bin:$PATH
export GOPATH=$(pwd)/gopath

cd gopath/src/github.com/cloudfoundry/bosh-agent
bin/test-integration-windows
