#!/bin/bash

exec fly -t production set-pipeline -p bosh:cli -c ./pipeline.yml --load-vars-from <(lpass show -G "bosh-cli concourse secrets" --notes)
