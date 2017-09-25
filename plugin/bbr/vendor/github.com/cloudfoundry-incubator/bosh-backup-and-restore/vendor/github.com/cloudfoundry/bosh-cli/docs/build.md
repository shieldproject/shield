## Development Workstation Setup

Note: This guide assumes a few things:

- You have gcc (or an equivalent)
- You can install packages (brew, apt-get, or equivalent)

Get Golang and its dependencies (Mac example, replace with your package manager of choice):

- `brew update`
- `brew install go`
- `brew install git` (Go needs git for the `go get` command)
- `brew install hg` (Go needs mercurial for the `go get` command)

Clone and set up the repository:

- `go get -d github.com/cloudfoundry/bosh-cli`
- `cd $GOPATH/src/github.com/cloudfoundry/bosh-cli`

From here on out we assume you're working in `$GOPATH/src/github.com/cloudfoundry/bosh-cli`

To build the bosh-init cli:

- `bin/build` # The `bosh-init` binary will be located in `out/`
