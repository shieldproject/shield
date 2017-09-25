## Unit Tests

Each package in the CLI has its own unit tests and there are integration tests in the `integration` package.

You can also run all tests with `bin/test`.

## Acceptance Tests

The acceptance tests are designed to exercise the main commands of the CLI (deployment, deploy, delete).

They are not designed to verify the compatibility of CPIs or testing BOSH releases.

The acceptance test related to compiled releases uses an already compiled release that was compiled against a stemcell 
with os/version : ubuntu-trusty/2776. If the stemcell os/version used for the tests changes, you will need to modify the 
file [acceptance/assets/sample-release-compiled.tgz](acceptance/assets/sample-release-compiled.tgz). This can be either 
modified manually by un-zipping it and changing the release manifest, or you can refer build the sample release from 
[acceptance/assets/sample-release](acceptance/assets/sample-release) folder, upload it to a bosh installation, and 
deploy it against a stemcell with the desired OS and Version. Then use bosh export to export a compiled release.

### Fly executing the acceptance tests

In theory you should be able to export the environment variables for the task,
but I've had trouble getting that to work.

A way to run the acceptance tests that seems to work:

We're going to `fly execute` the test-acceptance.yml task by editing it directly
with our secrets. To make it harder to accidently check these secrets in, we
should copy that file somewhere like `/tmp/bosh-init-acceptance`. Now insert
real values for the environment variables so test-acceptance.yml looks something
like

```
...

params:
  BOSH_AWS_ACCESS_KEY_ID:     "ASDF"
  BOSH_AWS_SECRET_ACCESS_KEY: "asdfasdf"
  BOSH_LITE_KEYPAIR:          bosh-dev
  BOSH_LITE_SUBNET_ID:        subnet-12345
  BOSH_LITE_SECURITY_GROUP:   sg-5678
  BOSH_LITE_PRIVATE_KEY_DATA: |
    -----BEGIN RSA PRIVATE KEY-----
    sosecretandsecure
    -----END RSA PRIVATE KEY-----
```

We're also going to need all the inputs for the task. `bosh-init` is easy,
that's going to be the bosh-init source directory. To satisfy the
`bosh-warden-cpi-release` input, we'll need to download the warden cpi release
(probably from bosh.io) and name it `cpi-release.tgz`.

Now we can fly execute:

```
./fly -t bosh-init -k execute -p -c <path-to-test-acceptance.yml> -i bosh-init=<path-to-source-dir> -i bosh-warden-cpi-release=<path-to-dir-containing-cpi-release.tgz>
```

### Running the acceptance tests directly

#### Dependencies

- [Vagrant](https://www.vagrantup.com/)

    `brew install vagrant`

- [sshpass](http://linux.die.net/man/1/sshpass)

    `brew install https://raw.github.com/eugeneoden/homebrew/eca9de1/Library/Formula/sshpass.rb`

#### Providers

Acceptance tests can be run in a VM with the following vagrant providers:

* [virtualbox](https://www.virtualbox.org/) (free)
* [aws](http://aws.amazon.com/)

##### Local Provider

The acceptance tests can be run on a local VM (using Virtual Box with vagrant).

The acceptance tests require a stemcell and a BOSH Warden CPI release.

Without specifying them, a specific (known to work) version of each will be downloaded.

You may alternatively choose to download them to a local directory and specify their paths via environment variables. They will then be scp'd onto the vagrant VM.

To take advantage of this feature, export the following variables prior to running the tests (absolute paths are required):

```
$ export BOSH_INIT_CPI_RELEASE_PATH=/tmp/bosh-warden-cpi-9.tgz
$ export BOSH_INIT_STEMCELL_PATH=/tmp/bosh-stemcell-348-warden-boshlite-ubuntu-trusty-go_agent.tgz
$ ./bin/test-acceptance-with-vm --provider=virtualbox
```

You can use remote releases and stemcells which you can overwrite using environment variables below. In this case you also need to provide sha1 of remote artifacts you want to test.

```
$ export BOSH_INIT_CPI_RELEASE_URL=https://bosh.io/d/github.com/cppforlife/bosh-warden-cpi-release?v=6
$ export BOSH_INIT_CPI_RELEASE_SHA1=cpi-release-sha1
$ export BOSH_INIT_STEMCELL_URL=https://bosh.io/d/stemcells/bosh-warden-boshlite-ubuntu-trusty-go_agent?v=2776
$ export BOSH_INIT_STEMCELL_SHA1=stemcell-sha1
$ ./bin/test-acceptance-with-vm --provider=virtualbox
```

##### AWS Provider

The acceptance tests can also be run on a remote VM (using aws with vagrant).

When using the AWS provider, you will need to provide the following:

```
export BOSH_INIT_PRIVATE_KEY=~/tmp/bosh-dev.key

# The following variables are required by Bosh Lite
export BOSH_AWS_ACCESS_KEY_ID=foo
export BOSH_AWS_SECRET_ACCESS_KEY=bar
export BOSH_LITE_KEYPAIR=bosh-dev
export BOSH_LITE_SUBNET_ID=subnet-1234
export BOSH_LITE_NAME=baz
export BOSH_LITE_SECURITY_GROUP=sg-1234
export BOSH_LITE_PRIVATE_KEY=$BOSH_INIT_PRIVATE_KEY
```

##### Running tests against existing VM

Acceptance tests use configuration file specified via `BOSH_INIT_CONFIG_PATH`. The format of the configuration file is basic JSON.

```
{
  "vm_username": "TEST_VM_USERNAME",
  "vm_ip": "TEST_VM_IP",
  "private_key_path": "TEST_VM_PRIVATE_KEY_PATH",

  "cpi_release_url": "CPI_RELEASE_URL",
  "cpi_release_sha1": "CPI_RELEASE_SHA1",

  "stemcell_url": "STEMCELL_URL",
  "stemcell_sha1":"STEMCELL_SHA1",

  "dummy_release_path": "DUMMY_RELEASE_PATH"
}
```

Run acceptance tests:

```
BOSH_INIT_CONFIG_PATH=config.json bin/test-acceptance
```

## Debugging Acceptance Test Failures

If your acceptance tests are failing mysteriously while running a command, here are some things to check:

 * `vagrant ssh` to the vm running the specs and check out the `bosh-init.log` in the vagrant user home directory
 * If your agent isn't starting, get its IP from the bosh-init logs (see above). Then you can `ssh vcap@<ip>` and check out `/var/vcap/bosh/log/current`
