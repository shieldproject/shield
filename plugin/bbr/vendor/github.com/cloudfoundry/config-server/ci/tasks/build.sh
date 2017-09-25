#!/bin/sh
set -e -x

export GOPATH=$(pwd)
export PATH=/usr/local/go/bin:$GOPATH/bin:$PATH

semver=`cat version-semver/number`
filename="config-server-${semver}-${GOOS}-${GOARCH}"

go build github.com/cloudfoundry/config-server

mv config-server compiled-${GOOS}/${filename}

openssl sha256 compiled-${GOOS}/${filename}
