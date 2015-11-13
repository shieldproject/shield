package supervisor_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
	"os"
	"fmt"
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
