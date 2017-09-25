#!/usr/bin/env bash

set -e

export BASE=$(pwd)
export PATH=/usr/local/ruby/bin:/usr/local/go/bin:$PATH
export GOPATH=${BASE}/gopath

semver=`cat ${BASE}/version-semver/number`

filename="bosh-agent-${semver}-${GOOS}-${GOARCH}"
if [[ $GOOS = 'windows' ]]; then
  filename="${filename}.exe"
fi

timestamp=`date -u +"%Y-%m-%dT%H:%M:%SZ"`
go_ver=`go version | cut -d ' ' -f 3`

cd gopath/src/github.com/cloudfoundry/bosh-agent

git_rev=`git rev-parse --short HEAD`

version="${semver}-${git_rev}-${timestamp}-${go_ver}"
sed -i 's/\[DEV BUILD\]/'"$version"'/' main/version.go

bin/build

shasum -a 256 out/bosh-agent

cp out/bosh-agent "${BASE}/compiled-${GOOS}-${GOARCH}/${filename}"