package blobstore_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCascadingBlobstore(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cascading Blobstore Suite")
}
