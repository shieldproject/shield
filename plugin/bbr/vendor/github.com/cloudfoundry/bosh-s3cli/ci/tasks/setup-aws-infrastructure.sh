#!/usr/bin/env bash

set -e

my_dir="$( cd $(dirname $0) && pwd )"
release_dir="$( cd ${my_dir} && cd ../.. && pwd )"
workspace_dir="$( cd ${release_dir} && cd ../../../.. && pwd )"

source ${release_dir}/ci/tasks/utils.sh
export GOPATH=${workspace_dir}
export PATH=${GOPATH}/bin:${PATH}

: ${access_key_id:?}
: ${secret_access_key:?}
: ${region_name:?}
: ${stack_name:?}

export AWS_ACCESS_KEY_ID=${access_key_id}
export AWS_SECRET_ACCESS_KEY=${secret_access_key}
export AWS_DEFAULT_REGION=${region_name}

cmd="aws cloudformation create-stack \
    --stack-name    ${stack_name} \
    --template-body file://${release_dir}/ci/assets/cloudformation-${stack_name}.template.json \
    --capabilities  CAPABILITY_IAM"
echo "Running: ${cmd}"; ${cmd}

while true; do
  stack_status=$(get_stack_status $stack_name)
  echo "StackStatus ${stack_status}"
  if [ $stack_status == 'CREATE_IN_PROGRESS' ]; then
    echo "sleeping 5s"; sleep 5s
  else
    break
  fi
done

if [ $stack_status != 'CREATE_COMPLETE' ]; then
  echo "cloudformation failed stack info:\n$(get_stack_info $stack_name)"
  exit 1
fi
