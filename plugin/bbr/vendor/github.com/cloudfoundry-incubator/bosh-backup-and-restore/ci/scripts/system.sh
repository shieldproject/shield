#!/bin/bash

set -ex

eval "$(ssh-agent)"
./bosh-backup-and-restore-meta/unlock-ci.sh
chmod 400 bosh-backup-and-restore-meta/keys/github
chmod 400 bosh-backup-and-restore-meta/genesis-bosh/bosh.pem
ssh-add bosh-backup-and-restore-meta/keys/github
export BOSH_GATEWAY_HOST=$BOSH_HOST
export BOSH_URL=https://$BOSH_HOST
export GOPATH=$PWD
export PATH=$PATH:$GOPATH/bin
export BOSH_CERT_PATH=`pwd`/bosh-backup-and-restore-meta/certs/$BOSH_HOST.crt
export BOSH_GATEWAY_KEY=`pwd`/bosh-backup-and-restore-meta/genesis-bosh/bosh.pem

cd src/github.com/cloudfoundry-incubator/bosh-backup-and-restore
make sys-test-ci
