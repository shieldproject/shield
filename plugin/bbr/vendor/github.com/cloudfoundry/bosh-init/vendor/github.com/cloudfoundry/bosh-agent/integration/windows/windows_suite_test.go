package windows_test

import (
	"fmt"
	"os"

	"github.com/cloudfoundry/bosh-agent/integration/windows/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestWindows(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Windows Suite")
}

var _ = BeforeSuite(func() {
	vagrantProvider := os.Getenv("VAGRANT_PROVIDER")

	_, err := utils.StartVagrant(vagrantProvider)
	if err != nil {
		Fail(fmt.Sprintln("Could not build the bosh-agent project.\nError is:", err))
	}
})
