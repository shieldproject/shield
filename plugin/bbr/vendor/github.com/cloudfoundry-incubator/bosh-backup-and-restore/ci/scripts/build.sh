#!/bin/bash

set -ex

eval "$(ssh-agent)"
./bosh-backup-and-restore-meta/unlock-ci.sh
chmod 400 bosh-backup-and-restore-meta/keys/github
ssh-add bosh-backup-and-restore-meta/keys/github
export GOPATH=$PWD
export PATH=$PATH:$GOPATH/bin
export VERSION=$(cat version/number)

pushd src/github.com/cloudfoundry-incubator/bosh-backup-and-restore
  make release
  tar -cvf bbr-"$VERSION".tar releases/*
popd

mv src/github.com/cloudfoundry-incubator/bosh-backup-and-restore/bbr-"$VERSION".tar bbr-build/

echo "Auto-delivered in
https://s3-eu-west-1.amazonaws.com/bosh-backup-and-restore-builds/bbr-$VERSION.tar

[Backup and Restore Bot]" > bbr-build/message
