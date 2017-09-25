#!/usr/bin/env bash

set -ex

echo "Running ci/run-acceptance-with-vm-in-container.sh"
echo "ENV:"
echo `env`

BOSH_INIT_CLI_DIR=/home/ubuntu/go/src/github.com/cloudfoundry/bosh-cli

#inside the docker container
BOSH_INIT_PRIVATE_KEY_DIR=/home/ubuntu/private_keys
PRIVATE_KEY_DIR=$(cd $(dirname $BOSH_LITE_PRIVATE_KEY) && pwd)
BOSH_LITE_PRIVATE_KEY_BASENAME=$(basename $BOSH_LITE_PRIVATE_KEY)
BOSH_LITE_PRIVATE_KEY=${BOSH_INIT_PRIVATE_KEY_DIR}/${BOSH_LITE_PRIVATE_KEY_BASENAME}

echo "ENV:"
echo `env`

# Pushing to Docker Hub requires login
DOCKER_IMAGE=${DOCKER_IMAGE:-bosh/cli}

# To push to the Pivotal GoCD Docker Registry (behind firewall):
# DOCKER_IMAGE=docker.gocd.cf-app.com:5000/bosh-init-container

SRC_DIR=$(cd $(dirname $0)/.. && pwd)
chmod -R o+w $SRC_DIR

echo "Running '$@' in docker container '$DOCKER_IMAGE'..."

docker pull $DOCKER_IMAGE

docker run \
  -e BOSH_AWS_ACCESS_KEY_ID \
  -e BOSH_AWS_SECRET_ACCESS_KEY \
  -e BOSH_LITE_KEYPAIR \
  -e BOSH_LITE_SUBNET_ID \
  -e BOSH_LITE_NAME \
  -e BOSH_LITE_SECURITY_GROUP \
  -e BOSH_LITE_PRIVATE_KEY \
  -e BOSH_INIT_STEMCELL_URL \
  -e BOSH_INIT_CPI_RELEASE_URL \
  -v $SRC_DIR:$BOSH_INIT_CLI_DIR \
  -v $PRIVATE_KEY_DIR:$BOSH_INIT_PRIVATE_KEY_DIR \
  $DOCKER_IMAGE \
  $BOSH_INIT_CLI_DIR/bin/test-acceptance-with-vm --provider=aws \
  &

SUBPROC="$!"

trap "
  echo '--------------------- KILLING PROCESS'
  kill $SUBPROC

  echo '--------------------- KILLING CONTAINERS'
  docker ps -q | xargs docker kill
" SIGTERM SIGINT # gocd sends TERM; INT just nicer for testing with Ctrl+C

wait $SUBPROC
