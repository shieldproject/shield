#!/bin/bash
set -ex

status=$(vagrant status)
if echo $status | grep running | grep virtualbox
then
	echo "Vagrant is already running with a different provider"
	exit 1
fi

if echo $status | grep agent | grep running
then
  vagrant provision
else
  vagrant up --provider=aws
fi
