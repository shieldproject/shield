#!/usr/bin/env bash

get_stack_info() {
  local stack_name=$1

  echo "$(aws cloudformation describe-stacks)" | \
  jq --arg stack_name ${stack_name} '.Stacks[] | select(.StackName=="\($stack_name)")'
}

get_stack_info_of() {
  local stack_info=$1
  local key=$2
  echo "${stack_info}" | jq -r --arg key ${key} '.Outputs[] | select(.OutputKey=="\($key)").OutputValue'
}

get_stack_status() {
  local stack_name=$1

  local stack_info=$(get_stack_info $stack_name)
  echo "${stack_info}" | jq -r '.StackStatus'
}
