#!/usr/bin/env bash

set -ex

GOPATH=/home/vagrant/go
export GOROOT=/usr/local/go
export PATH=$GOROOT/bin:$PATH
GO_ARCHIVE_URL=https://storage.googleapis.com/golang/go1.5.1.linux-amd64.tar.gz
GO_ARCHIVE=$GOPATH/src/github.com/cloudfoundry/bosh-agent/tmp/$(basename $GO_ARCHIVE_URL)

echo "Downloading go..."
mkdir -p $(dirname $GOROOT)
mkdir -p $GOPATH/src/github.com/cloudfoundry/bosh-agent/tmp
wget -q $GO_ARCHIVE_URL -O $GO_ARCHIVE
tar xf $GO_ARCHIVE -C $(dirname $GOROOT)
chmod -R a+w $GOROOT

if [ ! -d $TMPDIR ]; then
  mkdir -p $TMPDIR
fi

rm -f $GO_ARCHIVE