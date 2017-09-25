#!/usr/bin/env bash

set -e

my_dir="$( cd $(dirname $0) && pwd )"
release_dir="$( cd ${my_dir} && cd ../.. && pwd )"
workspace_dir="$( cd ${release_dir} && cd ../../../.. && pwd )"

export GOPATH=${workspace_dir}
export PATH=${GOPATH}/bin:${PATH}

# inputs
semver_dir="${workspace_dir}/version-semver"

# outputs
output_dir=${workspace_dir}/out

semver="$(cat ${semver_dir}/number)"
timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

binname="davcli-${semver}-${GOOS}-${GOARCH}"
if [ $GOOS = "windows" ]; then
    binname="${binname}.exe"
fi

pushd ${release_dir} > /dev/null
  git_rev=`git rev-parse --short HEAD`
  version="${semver}-${git_rev}-${timestamp}"

  echo -e "\n building artifact..."
  go build -ldflags "-X main.version=${version}" \
    -o "out/${binname}" \
    github.com/cloudfoundry/bosh-davcli/main

  echo -e "\n sha1 of artifact..."
  sha1sum "out/${binname}"

  mv "out/${binname}" ${output_dir}/
popd > /dev/null
