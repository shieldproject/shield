dummy-too-boshrelease
=================

A very simple [BOSH](https://github.com/cloudfoundry/bosh) release.

This exists to test that multiple releases can be co-located on one instance.

It has one job template:

1. `dummyToo` has no packages and only 1 job that monitors pid 1.
