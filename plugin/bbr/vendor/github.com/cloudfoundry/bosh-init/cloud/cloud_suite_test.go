package cloud_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestCloud(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cloud Suite")
}
