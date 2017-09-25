#!/usr/bin/env bash

set -ex

PROVIDER=${1:-virtualbox}
echo "vagrant provider: $PROVIDER"

vagrant up --provider=${PROVIDER} --provision
