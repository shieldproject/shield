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

stack_info=$(get_stack_info ${stack_name})
bucket_name=$(get_stack_info_of "${stack_info}" "BucketName")
aws s3 rm s3://${bucket_name} --recursive

cmd="aws cloudformation delete-stack --stack-name ${stack_name}"
echo "Running: ${cmd}"; ${cmd}

while true; do
  stack_status=$(get_stack_status $stack_name)
  echo "StackStatus ${stack_status}"
  if [[ -z "$stack_status" ]]; then #get empty status due to stack not existed on aws
    echo "No stack found"; break
    break
  elif [ $stack_status == 'DELETE_IN_PROGRESS' ]; then
    echo "${stack_status}: sleeping 5s"; sleep 5s
  else
    echo "Expecting the stack to either be deleted or in the process of being deleted but was ${stack_status}"
    echo $(get_stack_info $stack_name)
    exit 1
  fi
done
