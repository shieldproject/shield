# bosh-davcli

A CLI utility the BOSH Agent uses for accessing the [DAV blobstore](https://bosh.io/docs/director-configure-blobstore.html). 

Inside stemcells this binary is on the PATH as `bosh-blobstore-dav`.

### Developers

To update dependencies, use `gvt update`. Here is a typical invocation to update the `bosh-utils` dependency:

```
gvt update github.com/cloudfoundry/bosh-utils
```