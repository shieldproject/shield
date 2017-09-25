#!/bin/bash
mkdir -p micro_bosh/data/cache/
mkdir -p bosh
../out/bosh-agent  -b $PWD -P dummy -M dummy-nats -M dummy -C agent.json
