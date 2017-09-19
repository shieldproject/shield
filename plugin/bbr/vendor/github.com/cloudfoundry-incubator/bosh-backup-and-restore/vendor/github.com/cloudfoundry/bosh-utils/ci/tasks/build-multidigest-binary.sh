#!/bin/bash

set -e -x -u

base=$(pwd)
export GOPATH=$(pwd)/gopath
export PATH=/usr/local/ruby/bin:/usr/local/go/bin:$PATH


semver=`cat version-semver/number`
timestamp=`date -u +"%Y-%m-%dT%H:%M:%SZ"`
filename="verify-multidigest-${semver}-${GOOS}-${GOARCH}"

cd gopath/src/github.com/cloudfoundry/bosh-utils

bin/require-ci-golang-version

git_rev=`git rev-parse --short HEAD`
version="${semver}-${git_rev}-${timestamp}"

echo "building ${filename} with version ${version}"
sed -i 's/\[DEV BUILD\]/'"$version"'/' main/version.go

bin/build
shasum -a 256 out/verify-multidigest

mv out/verify-multidigest $base/out/${filename}
