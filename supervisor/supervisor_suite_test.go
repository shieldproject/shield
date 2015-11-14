package supervisor_test

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"os"
	"testing"
)

func TestSupervisor(t *testing.T) {
	RegisterFailHandler(Fail)

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	os.Setenv("PATH", fmt.Sprintf("%s:%s/../bin", os.Getenv("PATH"), wd))

	RunSpecs(t, "Supervisor Test Suite")
}
