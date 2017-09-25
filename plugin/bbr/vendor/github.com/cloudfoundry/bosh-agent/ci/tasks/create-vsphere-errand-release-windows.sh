#!/bin/bash

set -e -x


cd vsphere-errand-release
bosh init release $RELEASE_NAME

pushd $RELEASE_NAME
  bosh generate job $JOB_NAME

  cat > ./jobs/$JOB_NAME/templates/run.ps1 <<EOF
Write-Host "I am executing a simple bosh errand"
Get-ChildItem
EOF

  cat > ./jobs/$JOB_NAME/spec <<EOF
---
name: $JOB_NAME
description: "This is a simple errand"
templates:
  run.ps1: bin/run.ps1
EOF

   bosh create release --name $RELEASE_NAME --force --with-tarball --timestamp-version
popd


cat > ./manifest.yml <<EOF
---
name: $DEPLOYMENT_NAME
director_uuid: $DIRECTOR_UUID

releases:
- name: $RELEASE_NAME
  version: latest

networks:
- name: default
  subnets:
  - range: $BOSH_RANGE
    gateway: $BOSH_GATEWAY
    reserved: $BOSH_RESERVED
    static: $BOSH_STATIC
    dns: $BOSH_DNS
    cloud_properties:
      name: $BOSH_NETWORK_NAME

resource_pools:
- name: default
  stemcell:
    name: bosh-vsphere-esxi-windows-2012R2-go_agent
    version: latest
  network: default
  cloud_properties:
    cpu: 2
    ram: 2_048
    disk: 2_048


compilation:
  workers: 1
  network: default
  resource_pool: default
  cloud_properties:
    cpu: 2
    ram: 2_048
    disk: 2_048

update:
  canaries: 0
  canary_watch_time: 60000
  update_watch_time: 60000
  max_in_flight: 2

jobs:
- name: $JOB_NAME
  templates:
  - name: $JOB_NAME
  instances: 1
  resource_pool: default
  lifecycle: errand
  networks:
  - name: default
    static_ips: $BOSH_STATIC
  properties: {}
EOF
