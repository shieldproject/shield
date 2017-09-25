#!/usr/bin/env bash

set -ex

GOPATH=/home/vagrant/go
export GOROOT=/usr/local/go
export PATH=$GOROOT/bin:$PATH

echo "Installing bosh-agent..."
mkdir -p $(dirname $GOROOT)
chmod -R a+w $GOROOT

if [ ! -d $TMPDIR ]; then
  mkdir -p $TMPDIR
fi

agent_dir=$GOPATH/src/github.com/cloudfoundry/bosh-agent

pushd $agent_dir
	sudo sv stop agent

	# bosh-provisioner sends an apply spec that is no longer compatible with the agent
	sudo rm -f /var/vcap/bosh/spec.json

	# build agent
	bin/build

	# install new agent
	sudo cp out/bosh-agent /var/vcap/bosh/bin/bosh-agent
popd