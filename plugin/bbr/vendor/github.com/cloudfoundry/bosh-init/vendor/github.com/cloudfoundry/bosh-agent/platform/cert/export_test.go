package cert

/*
This is a workaround to make the private functions in CertManager available for
testing, the Go lang standard library include the following similar example:

In net http there are tests in both the http package (ie: http://golang.org/src...)
and the http_test package (ie: http://golang.org/src.... In order for the tests in
the http_test package to gain access to private functions for testing, there is an
export_test.go file in the http package that exports private items specifically for
testing. http://golang.org/src...

Because this is a *_test file it will not be included when you build the package,
but it will be available when running go test.
*/

import (
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

func SplitCerts(certs string) []string {
	return splitCerts(certs)
}

func DeleteFiles(fs boshsys.FileSystem, path string, filenamePrefix string) (int, error) {
	return deleteFiles(fs, path, filenamePrefix)
}
