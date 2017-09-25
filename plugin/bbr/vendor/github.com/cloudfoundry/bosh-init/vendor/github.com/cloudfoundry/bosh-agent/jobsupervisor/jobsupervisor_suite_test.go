package jobsupervisor_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestJobsupervisor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Job Supervisor Suite")
}
