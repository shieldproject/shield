#!/bin/bash

set -e

my_dir="$( cd "$( dirname "$0" )" && pwd )"

DOCKER_IMAGE=${DOCKER_IMAGE:-boshcpi/s3cli}

docker login

echo "Building docker image..."
docker build -t $DOCKER_IMAGE "${my_dir}"

echo "Pushing docker image to '$DOCKER_IMAGE'..."
docker push $DOCKER_IMAGE
