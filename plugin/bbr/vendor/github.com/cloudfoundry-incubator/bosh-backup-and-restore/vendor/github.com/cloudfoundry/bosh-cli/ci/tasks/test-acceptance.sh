#!/usr/bin/env bash

set -e -x

ensure_not_replace_value() {
  local name=$1
  local value=$(eval echo '$'$name)
  if [ "$value" == 'replace-me' ]; then
    echo "environment variable $name must be set"
    exit 1
  fi
}

set +x
if [[ `whoami` != "root" ]]; then
  echo "acceptance tests must be run as a privileged user"
  exit 1
fi

ensure_not_replace_value BOSH_AWS_ACCESS_KEY_ID
ensure_not_replace_value BOSH_AWS_SECRET_ACCESS_KEY
ensure_not_replace_value BOSH_LITE_KEYPAIR
ensure_not_replace_value BOSH_LITE_SUBNET_ID
ensure_not_replace_value BOSH_LITE_SECURITY_GROUP
ensure_not_replace_value BOSH_LITE_PRIVATE_KEY_DATA
set -x

export PATH=/usr/local/ruby/bin:/usr/local/go/bin:$PATH
export GOPATH=$PWD/gopath

export BOSH_INIT_CPI_RELEASE_PATH=`ls $PWD/cpi-release/*.tgz`
export BOSH_INIT_CPI_RELEASE_URL=""
export BOSH_INIT_CPI_RELEASE_SHA1=""

set +x
tmpfile=`mktemp -t bosh-init-tests-XXXXXXXX`
echo "$BOSH_LITE_PRIVATE_KEY_DATA" > $tmpfile
set -x

export BOSH_LITE_PRIVATE_KEY=$tmpfile

cd $GOPATH/src/github.com/cloudfoundry/bosh-cli
source ci/docker/deps-golang-1.7.1
./bin/require-ci-golang-version
base=$PWD ./bin/test-acceptance-with-vm --provider=aws
