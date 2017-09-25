package integration

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"time"

	"github.com/cloudfoundry/bosh-agent/agentclient"
	"github.com/cloudfoundry/bosh-agent/integration/integrationagentclient"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	"github.com/cloudfoundry/bosh-utils/httpclient"
	"github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type TestEnvironment struct {
	cmdRunner        boshsys.CmdRunner
	currentDeviceNum int
	sshTunnelProc    boshsys.Process
	logger           logger.Logger
	agentClient      agentclient.AgentClient
	deviceMap        map[int]string
}

func NewTestEnvironment(
	cmdRunner boshsys.CmdRunner,
) *TestEnvironment {
	return &TestEnvironment{
		cmdRunner:        cmdRunner,
		currentDeviceNum: 2,
		logger:           logger.NewLogger(logger.LevelDebug),
		deviceMap:        make(map[int]string),
	}
}

func (t *TestEnvironment) SetupConfigDrive() error {
	deviceNum, err := t.AttachLoopDevice(10)
	if err != nil {
		return err
	}

	setupConfigDriveTemplate := `
sudo mkfs -t ext3 -m 1 -v %s
sudo e2label %s config-2
sudo rm -rf /tmp/config-drive
sudo mkdir /tmp/config-drive
sudo mount /dev/disk/by-label/config-2 /tmp/config-drive
sudo chown vagrant:vagrant /tmp/config-drive
sudo mkdir -p /tmp/config-drive/ec2/latest
sudo cp %s/meta-data.json /tmp/config-drive/ec2/latest/meta-data.json
sudo cp %s/user-data.json /tmp/config-drive/ec2/latest/user-data.json
sudo umount /tmp/config-drive
`
	setupConfigDriveScript := fmt.Sprintf(setupConfigDriveTemplate, t.deviceMap[deviceNum], t.deviceMap[deviceNum], t.assetsDir(), t.assetsDir())

	_, err = t.RunCommand(setupConfigDriveScript)
	return err
}

type byLen []string

func (a byLen) Len() int           { return len(a) }
func (a byLen) Less(i, j int) bool { return len(a[i]) > len(a[j]) }
func (a byLen) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func (t *TestEnvironment) DetachDevice(dir string) error {
	mountPoints, err := t.RunCommand(fmt.Sprintf(`sudo mount | grep "on %s" | cut -d ' ' -f 3`, dir))
	if err != nil {
		return err
	}

	mountPointsSlice := strings.Split(mountPoints, "\n")
	sort.Sort(byLen(mountPointsSlice))
	for _, mountPoint := range mountPointsSlice {
		if mountPoint != "" {
			t.RunCommand(fmt.Sprintf("sudo fuser -km %s", mountPoint))
			t.RunCommand(fmt.Sprintf("sudo umount %s", mountPoint))
		}
	}

	_, err = t.RunCommand(fmt.Sprintf("sudo rm -rf %s", dir))
	return err
}

func (t *TestEnvironment) CleanupDataDir() error {
	t.RunCommand(`sudo /var/vcap/bosh/bin/monit stop all`)

	_, err := t.RunCommand("! mount | grep -q ' on /tmp ' || sudo umount /tmp")
	if err != nil {
		return err
	}

	err = t.DetachDevice("/var/tmp")
	if err != nil {
		return err
	}

	err = t.DetachDevice("/var/log")
	if err != nil {
		return err
	}

	err = t.DetachDevice("/var/vcap/data")
	if err != nil {
		return err
	}

	_, err = t.RunCommand("sudo mkdir -p /var/tmp")
	if err != nil {
		return err
	}

	_, err = t.RunCommand("sudo chmod 700 /var/tmp")
	if err != nil {
		return err
	}

	_, err = t.RunCommand("sudo chmod 1777 /tmp")
	if err != nil {
		return err
	}

	_, err = t.RunCommand("sudo mkdir -p /var/log")
	if err != nil {
		return err
	}

	_, err = t.RunCommand("sudo chmod 775 /var/log")
	if err != nil {
		return err
	}

	_, err = t.RunCommand("sudo chown root:syslog /var/log")
	if err != nil {
		return err
	}

	return nil
}

// ConfigureAgentForGenericInfrastructure executes the agent_runit.sh asset.
// Required for reverse-compatibility with older bosh-lite
// (remove once a new warden stemcell is built).
func (t *TestEnvironment) ConfigureAgentForGenericInfrastructure() error {
	_, err := t.RunCommand(
		fmt.Sprintf(
			"sudo cp %s/agent_runit.sh /etc/service/agent/run",
			t.assetsDir(),
		),
	)
	return err
}

func (t *TestEnvironment) CleanupLogFile() error {
	_, err := t.RunCommand("sudo truncate -s 0 /var/vcap/bosh/log/current")
	return err
}

func (t *TestEnvironment) CleanupSSH() error {
	_, err := t.RunCommand("sudo rm -rf /var/vcap/bosh_ssh")
	return err
}

func (t *TestEnvironment) LogFileContains(content string) bool {
	_, err := t.RunCommand(fmt.Sprintf(`sudo grep "%s" /var/vcap/bosh/log/current`, content))
	return err == nil
}

func (t *TestEnvironment) AttachDevice(devicePath string, partitionSize, numPartitions int) error {
	partitionPath := devicePath
	for i := 0; i <= numPartitions; i++ {
		if i > 0 {
			partitionPath = fmt.Sprintf("%s%d", devicePath, i)
		}

		deviceNum, err := t.AttachLoopDevice(partitionSize)
		if err != nil {
			return err
		}

		output, err := t.RunCommand(fmt.Sprintf("ls -al %s | cut -d' ' -f 6", t.deviceMap[deviceNum]))
		minorNum := strings.TrimSpace(output)
		if err != nil {
			return err
		}

		err = t.RemoveDevice(partitionPath)
		if err != nil {
			return err
		}

		_, err = t.RunCommand(fmt.Sprintf("sudo mknod %s b 7 %s", partitionPath, minorNum))
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *TestEnvironment) AttachPartitionedRootDevice(devicePath string, sizeInMB, rootPartitionSizeInMB int) (string, error) {
	err := t.AttachDevice(devicePath, sizeInMB, 3)
	if err != nil {
		return "", err
	}

	// Create only first partition, agent will partition the rest for ephemeral disk
	partitionTemplate := `
echo ',%d,L,' | sudo sfdisk -uM %s
`
	partitionScript := fmt.Sprintf(partitionTemplate, rootPartitionSizeInMB, devicePath)
	_, err = t.RunCommand(partitionScript)
	if err != nil {
		return "", err
	}

	rootLink, err := t.RunCommand("df / | grep /dev/ | cut -d' ' -f1")
	if err != nil {
		return "", err
	}

	oldRootDevice, err := t.RunCommand(fmt.Sprintf("readlink -f %s", rootLink))
	if err != nil {
		return "", err
	}

	_, err = t.RunCommand(fmt.Sprintf("sudo mv %s %s-temp", strings.TrimSpace(oldRootDevice), strings.TrimSpace(oldRootDevice)))
	if err != nil {
		return "", err
	}

	// Agent reads the symlink to get root device
	// Create a symlink to our fake device
	_, err = t.RunCommand(fmt.Sprintf("sudo ln -sf %s1 %s", devicePath, strings.TrimSpace(rootLink)))

	if err != nil {
		return strings.TrimSpace(oldRootDevice), err
	}

	return strings.TrimSpace(oldRootDevice), nil
}

func (t *TestEnvironment) DetachPartitionedRootDevice(rootLink string, devicePath string) error {
	_, err := t.RunCommand(fmt.Sprintf("sudo rm -f %s", rootLink))
	if err != nil {
		return err
	}

	partitionPath := devicePath
	for i := 3; i >= 0; i-- {
		if i > 0 {
			partitionPath = fmt.Sprintf("%s%d", devicePath, i)
		}

		if _, err := t.RunCommand(fmt.Sprintf("losetup %s", partitionPath)); err == nil {
			if output, _ := t.RunCommand(fmt.Sprintf("sudo mount | grep '%s ' | awk '{print $3}'", partitionPath)); output != "" {
				t.RunCommand(fmt.Sprintf("sudo umount -l %s", output))
			}

			if i > 0 {
				_, _ = t.RunCommand(fmt.Sprintf("sudo parted %s rm %d", devicePath, i))
			}

			err = t.DetachLoopDevice(partitionPath)
			if err != nil {
				return err
			}

			err = t.RemoveDevice(partitionPath)
			if err != nil {
				return err
			}
		}

	}

	_, err = t.RunCommand(fmt.Sprintf("sudo mv %s-temp %s", rootLink, rootLink))
	if err != nil {
		return err
	}

	return nil
}

func (t *TestEnvironment) RemoveDevice(devicePath string) error {
	_, err := t.RunCommand(fmt.Sprintf("sudo rm -f %s", devicePath))
	return err
}

func (t *TestEnvironment) AttachLoopDevice(size int) (int, error) {
	deviceNum := t.currentDeviceNum

	output, err := t.RunCommand("sudo losetup -f")
	devicePath := strings.TrimSpace(output)
	if err != nil {
		return 0, err
	}

	if oldDevicePath, ok := t.deviceMap[deviceNum]; ok {
		t.DetachLoopDevice(oldDevicePath)
	}

	attachDeviceTemplate := `
sudo rm -rf /virtualfs-%d
sudo dd if=/dev/zero of=/virtualfs-%d bs=1M count=%d
sudo losetup %s /virtualfs-%d
`
	attachDeviceScript := fmt.Sprintf(attachDeviceTemplate, deviceNum, deviceNum, size, devicePath, deviceNum)
	_, err = t.RunCommand(attachDeviceScript)
	if err != nil {
		return 0, err
	}

	t.deviceMap[deviceNum] = devicePath
	t.currentDeviceNum++

	return deviceNum, nil
}

func (t *TestEnvironment) DetachLoopDevice(devicePath string) error {
	_, err := t.RunCommand(fmt.Sprintf("sudo losetup -d %s", devicePath))
	return err
}

func (t *TestEnvironment) UpdateAgentConfig(configFile string) error {
	_, err := t.RunCommand(
		fmt.Sprintf(
			"sudo cp %s/%s /var/vcap/bosh/agent.json",
			t.assetsDir(),
			configFile,
		),
	)
	return err
}

func (t *TestEnvironment) RestartAgent() error {
	err := t.StopAgent()
	if err != nil {
		return err
	}

	return t.StartAgent()
}

func (t *TestEnvironment) StopAgent() error {
	_, err := t.RunCommand("nohup sudo sv stop agent &")
	return err
}

func (t *TestEnvironment) StartAgent() error {
	_, err := t.RunCommand("nohup sudo sv start agent &")
	return err
}

type emptyReader struct{}

func (er emptyReader) Read(p []byte) (int, error) {
	time.Sleep(1 * time.Second)
	return 0, nil
}

func (t *TestEnvironment) StartAgentTunnel(mbusUser, mbusPass string, mbusPort int) (*integrationagentclient.IntegrationAgentClient, error) {
	if t.sshTunnelProc != nil {
		return nil, fmt.Errorf("Already running")
	}

	sshCmd := boshsys.Command{
		Name: "vagrant",
		Args: []string{
			"ssh",
			"--",
			fmt.Sprintf("-L16868:127.0.0.1:%d", mbusPort),
		},
		Stdin: emptyReader{},
	}
	newTunnelProc, err := t.cmdRunner.RunComplexCommandAsync(sshCmd)
	if err != nil {
		return nil, err
	}
	t.sshTunnelProc = newTunnelProc

	httpClient := httpclient.NewHTTPClient(httpclient.DefaultClient, t.logger)
	mbusURL := fmt.Sprintf("https://%s:%s@localhost:16868", mbusUser, mbusPass)
	client := integrationagentclient.NewIntegrationAgentClient(mbusURL, "fake-director-uuid", 1*time.Second, 10, httpClient, t.logger)

	for i := 1; i < 1000000; i++ {
		t.logger.Debug("test environment", "Trying to contact agent via ssh tunnel...")
		time.Sleep(1 * time.Second)
		_, err := client.Ping()
		if err == nil {
			break
		}
		t.logger.Debug("test environment", err.Error())
	}
	return client, nil
}

func (t *TestEnvironment) StopAgentTunnel() error {
	if t.sshTunnelProc == nil {
		return fmt.Errorf("Not running")
	}
	t.sshTunnelProc.Wait()
	t.sshTunnelProc = nil
	return nil
}

func (t *TestEnvironment) StartRegistry(settings boshsettings.Settings) error {
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return err
	}

	_, err = t.RunCommand("sudo rm -f /var/vcap/bosh/settings.json")
	if err != nil {
		return err
	}

	t.RunCommand("sudo killall -9 fake-registry")

	_, err = t.RunCommand(
		fmt.Sprintf(
			`nohup %s/tmp/fake-registry -user user -password pass -host 127.0.0.1 -port 9090 -instance instance-id -settings %s &> /dev/null &`,
			t.agentDir(),
			strconv.Quote(string(settingsJSON)),
		),
	)
	return err
}

func (t *TestEnvironment) GetVMNetworks() (boshsettings.Networks, error) {
	stdout, _, _, err := t.cmdRunner.RunCommand("vagrant", "status")
	if err != nil {
		return boshsettings.Networks{}, err
	}

	if strings.Contains(stdout, "virtualbox") {
		return boshsettings.Networks{
			"eth0": {
				Type: "dynamic",
			},
			"eth1": {
				Type:    "manual",
				IP:      "192.168.50.4",
				Netmask: "255.255.255.0",
			},
		}, nil
	}

	if strings.Contains(stdout, "aws") {
		return boshsettings.Networks{
			"eth0": {
				Type: "dynamic",
			},
		}, nil
	}

	return boshsettings.Networks{}, nil
}

func (t *TestEnvironment) GetFileContents(filePath string) (string, error) {
	return t.RunCommand(
		fmt.Sprintf(
			`cat %s`,
			filePath,
		),
	)
}

func (t *TestEnvironment) RunCommand(command string) (string, error) {
	stdout, _, _, err := t.RunCommand3(command)
	return stdout, err
}

func (t *TestEnvironment) RunCommand3(command string) (string, string, int, error) {
	return t.cmdRunner.RunCommand("vagrant", "ssh", "-c", command)
}

func (t *TestEnvironment) CreateBlobFromAsset(assetPath, blobID string) error {
	_, err := t.RunCommand("sudo mkdir -p /var/vcap/data")
	if err != nil {
		return err
	}

	_, _, _, err = t.cmdRunner.RunCommand(
		"vagrant",
		"ssh",
		"-c",
		fmt.Sprintf("sudo cp %s/%s /var/vcap/data/%s", t.assetsDir(), assetPath, blobID),
	)

	return err
}

func (t *TestEnvironment) agentDir() string {
	return "/home/vagrant/go/src/github.com/cloudfoundry/bosh-agent"
}

func (t *TestEnvironment) assetsDir() string {
	return fmt.Sprintf("%s/integration/assets", t.agentDir())
}
