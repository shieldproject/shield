package system

import (
	"os/exec"
	"strings"

	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

type Deployment struct {
	Name     string
	Manifest string
}

type Instance struct {
	deployment Deployment
	Group      string
	Index      string
}

func NewDeployment(name, manifest string) Deployment {
	return Deployment{Name: name, Manifest: manifest}
}

func (d Deployment) Deploy() {
	session := d.runBosh("deploy", "--var=deployment-name="+d.Name, d.Manifest)
	Eventually(session).Should(gexec.Exit(0))
}

func (d Deployment) Delete() {
	session := d.runBosh("delete-deployment")
	Eventually(session).Should(gexec.Exit(0))
}

func (d Deployment) Instance(group, index string) Instance {
	return Instance{deployment: d, Group: group, Index: index}
}

func (d Deployment) runBosh(args ...string) *gexec.Session {
	boshCommand := fmt.Sprintf("bosh-cli --non-interactive --environment=%s --deployment=%s --ca-cert=%s --client=%s --client-secret=%s",
		MustHaveEnv("BOSH_URL"),
		d.Name,
		MustHaveEnv("BOSH_CERT_PATH"),
		MustHaveEnv("BOSH_CLIENT"),
		MustHaveEnv("BOSH_CLIENT_SECRET"),
	)

	return run(boshCommand, args...)
}

func run(cmd string, args ...string) *gexec.Session {
	cmdParts := strings.Split(cmd, " ")
	commandPath := cmdParts[0]
	combinedArgs := append(cmdParts[1:], args...)
	command := exec.Command(commandPath, combinedArgs...)

	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)

	Expect(err).ToNot(HaveOccurred())
	return session
}

func (i Instance) RunCommand(command string) *gexec.Session {
	return i.deployment.runBosh("ssh",
		"--gw-user="+MustHaveEnv("BOSH_GATEWAY_USER"),
		"--gw-host="+MustHaveEnv("BOSH_GATEWAY_HOST"),
		"--gw-private-key="+MustHaveEnv("BOSH_GATEWAY_KEY"),
		i.Group+"/"+i.Index,
		command)
}

func (i Instance) RunCommandAs(user, command string) *gexec.Session {
	return i.RunCommand(fmt.Sprintf(`sudo su vcap -c '%s'`, command))
}

func (i Instance) Copy(sourcePath, destinationPath string) {
	session := i.deployment.runBosh("scp",
		"--gw-user="+MustHaveEnv("BOSH_GATEWAY_USER"),
		"--gw-host="+MustHaveEnv("BOSH_GATEWAY_HOST"),
		"--gw-private-key="+MustHaveEnv("BOSH_GATEWAY_KEY"),
		sourcePath,
		i.Group+"/"+i.Index+":"+destinationPath,
	)
	Eventually(session).Should(gexec.Exit(0))
}

func (i Instance) AssertFilesExist(paths []string) {
	for _, path := range paths {
		cmd := i.RunCommandAs("vcap", "stat "+path)
		Eventually(cmd).Should(gexec.Exit(0), fmt.Sprintf("File at %s not found on %s/%s\n", path, i.Group, i.Index))
	}
}
