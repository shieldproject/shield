#!/usr/bin/env bash

set -e

my_dir="$( cd $(dirname $0) && pwd )"
release_dir="$( cd ${my_dir} && cd ../.. && pwd )"
workspace_dir="$( cd ${release_dir} && cd ../../../.. && pwd )"

export GOPATH=${workspace_dir}
export PATH=${GOPATH}/bin:${PATH}

: ${access_key_id:?}
: ${secret_access_key:?}
: ${bucket_name:?}
: ${s3_endpoint_host:?}
: ${s3_endpoint_port:?}

export ACCESS_KEY_ID=${access_key_id}
export SECRET_ACCESS_KEY=${secret_access_key}
export BUCKET_NAME=${bucket_name}
export S3_HOST=${s3_endpoint_host}
export S3_PORT=${s3_endpoint_port}

pushd ${release_dir} > /dev/null
  ginkgo -r -focus="S3 COMPATIBLE" integration/
popd > /dev/null
