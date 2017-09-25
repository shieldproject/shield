#!/usr/bin/env bash

set -e

my_dir="$( cd $(dirname $0) && pwd )"
release_dir="$( cd ${my_dir} && cd ../.. && pwd )"
workspace_dir="$( cd ${release_dir} && cd ../../../.. && pwd )"

export GOPATH=${workspace_dir}
export PATH=${GOPATH}/bin:${PATH}

semver='1.2.3.4'
timestamp=`date -u +"%Y-%m-%dT%H:%M:%SZ"`

pushd ${release_dir} > /dev/null
  git_rev=`git rev-parse --short HEAD`
  version="${semver}-${git_rev}-${timestamp}"

  DAVCLI_FILES=$(find . -type f -name '*.go' -not -path "*/vendor/*")

  echo -e "\n Vetting packages for potential issues..."
  for f in $DAVCLI_FILES ; do
    go vet $f
  done

  echo -e "\n Installing ginkgo..."
  go install ./vendor/github.com/onsi/ginkgo/ginkgo

  echo -e "\n Unit testing packages..."
  go get gopkg.in/yaml.v2
  ginkgo -r -race ./

  echo -e "\n suite success"
popd > /dev/null
