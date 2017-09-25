package action_test

import (
	"runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

const Windows = runtime.GOOS == "windows"

func TestAction(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Action Suite")
}
