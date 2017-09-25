#!/usr/bin/env bash

fly -t production set-pipeline \
    -p bosh-agent \
    -c pipeline.yml \
    --load-vars-from <(lpass show -G "bosh-agent concourse secrets" --notes)