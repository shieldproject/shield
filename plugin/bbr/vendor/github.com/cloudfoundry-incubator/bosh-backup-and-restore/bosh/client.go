package bosh

import (
	"strconv"

	"fmt"

	"github.com/cloudfoundry/bosh-cli/director"
	"github.com/cloudfoundry/bosh-utils/uuid"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/instance"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/orchestrator"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ssh"

	"github.com/pkg/errors"
	gossh "golang.org/x/crypto/ssh"
)

//go:generate counterfeiter -o fakes/fake_bosh_client.go . BoshClient
type BoshClient interface {
	FindInstances(deploymentName string) ([]orchestrator.Instance, error)
	GetManifest(deploymentName string) (string, error)
}

func NewClient(boshDirector director.Director,
	sshOptsGenerator ssh.SSHOptsGenerator,
	connectionFactory ssh.SSHConnectionFactory,
	logger Logger,
	jobFinder instance.JobFinder) Client {
	return Client{
		Director:             boshDirector,
		SSHOptsGenerator:     sshOptsGenerator,
		SSHConnectionFactory: connectionFactory,
		Logger:               logger,
		jobFinder:            jobFinder,
	}
}

type Client struct {
	director.Director
	ssh.SSHOptsGenerator
	ssh.SSHConnectionFactory
	Logger
	jobFinder instance.JobFinder
}

//go:generate counterfeiter -o fakes/fake_logger.go . Logger
type Logger interface {
	Debug(tag, msg string, args ...interface{})
	Info(tag, msg string, args ...interface{})
	Warn(tag, msg string, args ...interface{})
	Error(tag, msg string, args ...interface{})
}

func (c Client) FindInstances(deploymentName string) ([]orchestrator.Instance, error) {
	deployment, err := c.Director.FindDeployment(deploymentName)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't find deployment "+deploymentName)
	}

	c.Logger.Debug("bbr", "Finding VMs...")
	vms, err := deployment.VMInfos()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get vm infos")
	}
	sshOpts, privateKey, err := c.SSHOptsGenerator(uuid.NewGenerator())
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate ssh options")
	}
	c.Logger.Debug("bbr", "SSH user generated: %s", sshOpts.Username)

	instances := []orchestrator.Instance{}
	slugs := []director.AllOrInstanceGroupOrInstanceSlug{}

	for _, instanceGroupName := range uniqueInstanceGroupNamesFromVMs(vms) {
		c.Logger.Debug("bbr", "Setting up SSH for job %s", instanceGroupName)

		allVmInstances, err := director.NewAllOrInstanceGroupOrInstanceSlugFromString(instanceGroupName)
		if err != nil {
			cleanupAlreadyMadeConnections(deployment, slugs, sshOpts)
			return nil, errors.Wrap(err, "invalid instance group name: "+instanceGroupName)
		}

		sshRes, err := deployment.SetUpSSH(allVmInstances, sshOpts)
		if err != nil {
			cleanupAlreadyMadeConnections(deployment, slugs, sshOpts)
			return nil, errors.Wrap(err, "failed to set up ssh")
		}
		slugs = append(slugs, allVmInstances)

		for index, host := range sshRes.Hosts {
			var sshConnection ssh.SSHConnection
			var err error

			c.Logger.Debug("bbr", "Attempting to SSH onto %s, %s", host.Host, host.IndexOrID)

			hostPublicKey, _, _, _, err := gossh.ParseAuthorizedKey([]byte(host.HostPublicKey))
			if err != nil {
				return nil, errors.Wrap(err, "ssh.NewConnection.ParseAuthorizedKey failed")
			}

			sshConnection, err = c.SSHConnectionFactory(host.Host, host.Username, privateKey, gossh.FixedHostKey(hostPublicKey), []string{hostPublicKey.Type()}, c.Logger)
			if err != nil {
				cleanupAlreadyMadeConnections(deployment, slugs, sshOpts)
				return nil, errors.Wrap(err, "failed to connect using ssh")
			}

			hostIdentifier := fmt.Sprintf("%s/%s", instanceGroupName, host.IndexOrID)

			jobs, err := c.jobFinder.FindJobs(hostIdentifier, sshConnection)
			if err != nil {
				cleanupAlreadyMadeConnections(deployment, slugs, sshOpts)
				return nil, errors.Wrap(err, "couldn't find jobs")
			}

			instances = append(instances,
				NewBoshDeployedInstance(
					instanceGroupName,
					strconv.Itoa(index),
					host.IndexOrID,
					sshConnection,
					deployment,
					false,
					c.Logger,
					jobs,
				),
			)
		}
	}

	return instances, nil
}

func (c Client) GetManifest(deploymentName string) (string, error) {
	deployment, err := c.Director.FindDeployment(deploymentName)
	if err != nil {
		return "", errors.Wrap(err, "couldn't find deployment "+deploymentName)
	}
	return deployment.Manifest()
}

func uniqueInstanceGroupNamesFromVMs(vms []director.VMInfo) []string {
	var jobs []string
	for _, vm := range vms {
		if !contains(jobs, vm.JobName) {
			jobs = append(jobs, vm.JobName)
		}
	}
	return jobs
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func cleanupAlreadyMadeConnections(deployment director.Deployment, slugs []director.AllOrInstanceGroupOrInstanceSlug, opts director.SSHOpts) {
	for _, slug := range slugs {
		deployment.CleanUpSSH(slug, director.SSHOpts{Username: opts.Username})
	}
}
