package fmt_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestTime(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "UI Format Suite")
}
