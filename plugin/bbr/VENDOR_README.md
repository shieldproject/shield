This plugin has a custom vendor directory because godep doesn't play nice with BBR.

Additionally, BBR requires a newer version of the x/crypto/ssh library than SHIELD
currently supports.

To update the BBR deps:

1. `go get github.com/cloudfoundry-incubator/bosh-backup-and-restore`
2. `cd $GOPATH/github.com/cloudfoundry-incubator/bosh-backup-and-restore`
3. `make setup` # installs the bbr deps via glide
4. `cd` back to this directory
5. `rm -rf vendor`
6. `mkdir -p vendor/github.com/cloudfoundry-incubator
7. `cp -R $GOPATH/github.com/cloudfoundry-incubator/bosh-backup-and-restore vendor/github.com/cloudfoundry-incubator
8. Build + test
