#!/usr/bin/env bash

set -ex

source deps-golang-from-docker-files
GOROOT=/usr/local/go
GO_ARCHIVE=$(basename $GO_ARCHIVE_URL)

echo "Downloading go..."
mkdir -p $(dirname $GOROOT)
wget -q $GO_ARCHIVE_URL
echo "${GO_ARCHIVE_SHA256} ${GO_ARCHIVE}" | sha256sum -c -
tar xf $GO_ARCHIVE -C $(dirname $GOROOT)
chmod -R a+w $GOROOT
