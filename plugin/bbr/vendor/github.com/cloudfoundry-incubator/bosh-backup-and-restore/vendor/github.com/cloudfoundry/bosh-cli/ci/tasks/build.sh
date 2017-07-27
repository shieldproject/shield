#!/usr/bin/env bash

set -e -x -u

export PATH=/usr/local/ruby/bin:/usr/local/go/bin:$PATH
export GOPATH=$(pwd)/gopath

base=`pwd`

semver=`cat version-semver/number`
timestamp=`date -u +"%Y-%m-%dT%H:%M:%SZ"`
filename="${FILENAME_PREFIX}bosh-cli-${semver}-${GOOS}-${GOARCH}"

if [[ $GOOS = 'windows' ]]; then
  filename="${filename}.exe"
fi

cd gopath/src/github.com/cloudfoundry/bosh-cli
source ci/docker/deps-golang-1.7.1
bin/require-ci-golang-version

git_rev=`git rev-parse --short HEAD`
version="${semver}-${git_rev}-${timestamp}"

echo "building ${filename} with version ${version}"
sed 's/\[DEV BUILD\]/'"$version"'/' cmd/version.go > cmd/version.tmp && mv cmd/version{.tmp,.go}

bin/build

shasum_value=`sha1sum out/bosh | cut -f 1 -d' '`

set +x
if [[ $SKIP_BOSH_IO = false ]]; then
  curl --fail -X POST "https://bosh.io/checksums/${filename}" -d "sha1=${shasum_value}" -H "Authorization: bearer ${BOSHIO_BEARER_TOKEN}"
fi
set -x

mv out/bosh $base/compiled-${GOOS}/${filename}
