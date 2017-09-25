#!/usr/bin/env bash

set -ex

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
AGENT_OUTPUT_PATH="$DIR/fixtures/bosh-agent.exe"
PIPE_OUTPUT_PATH="$DIR/fixtures/pipe.exe"

rm -f "$AGENT_OUTPUT_PATH"
rm -f "$PIPE_OUTPUT_PATH"

GOOS=windows go build -o "$AGENT_OUTPUT_PATH" github.com/cloudfoundry/bosh-agent/main
GOOS=windows go build -o "$PIPE_OUTPUT_PATH" github.com/cloudfoundry/bosh-agent/jobsupervisor/pipe
