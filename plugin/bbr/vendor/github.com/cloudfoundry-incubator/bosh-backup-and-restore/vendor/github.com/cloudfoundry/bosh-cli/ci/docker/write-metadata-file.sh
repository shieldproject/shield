#!/usr/bin/env bash

set -e -x

source ~/.bashrc

echo $GO_VERSION | tee $(pwd)/docker-files/version
echo $GO_SHA | tee $(pwd)/docker-files/sha
echo $DOCKER_IMAGE_TAG | tee $(pwd)/docker-files/tag

cp bosh-cli-src/ci/docker/* $(pwd)/docker-files/

