# Repacking Stemcell with Updated Agent

!!! For debugging only !!!

This Concourse task allows the customization of a stemcell's `bosh-agent`, typically used when testing bootstrapping stage. Consider scp-ing Agent binary onto running machine as a quicker alternative.

Override STEMCELL_URL (must be heavy stemcell, raw) and STEMCELL_SHA1 to the stemcell you want (or leave as is):

```
export STEMCELL_URL=https://s3.amazonaws.com/bosh-core-stemcells/aws/bosh-stemcell-3312.15-aws-xen-ubuntu-trusty-go_agent.tgz
export STEMCELL_SHA1=d5252cdd6b07763ed989fcfeff47d06afa164065
./run.sh
```

Assumes your Concourse target is named _production_. If not, edit `run.sh` and adjust.

### Optional SSH debugging

Set your SSH key:

```
export BOSH_DEBUG_PUB_KEY="ssh-rsa blahblah"
```

This will bake your SSH public key into the stemcell; you will be able to ssh in as user `bosh_debug`.
