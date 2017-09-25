package utils

import (
	"os/exec"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const (
	agentID = "123-456-789"
)

type Agent struct {
	ID   string
	tail *gexec.Session
}

func BuildAgent() error {
	command := exec.Command("./build_agent.bash")
	session, err := gexec.Start(command, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	if err != nil {
		return err
	}
	gomega.Eventually(session, 40*time.Minute).Should(gexec.Exit(0))
	return nil
}

func StartVagrant(provider string) (Agent, error) {
	if len(provider) == 0 {
		provider = "virtualbox"
	}
	command := exec.Command("./setup_vagrant.bash", provider)
	session, err := gexec.Start(command, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	if err != nil {
		return Agent{}, err
	}
	gomega.Eventually(session, 40*time.Minute).Should(gexec.Exit(0))

	return Agent{
		ID: agentID,
	}, nil
}

func (a Agent) Stop() {
}
