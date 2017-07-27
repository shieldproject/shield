package stemcell_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestStemcell(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Stemcell Suite")
}
