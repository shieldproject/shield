package cert

import (
	"fmt"
	"os"
	"path"
	"strconv"

	boshdir "github.com/cloudfoundry/bosh-agent/settings/directories"
	"github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type windowsCertManager struct {
	fs          boshsys.FileSystem
	runner      boshsys.CmdRunner
	dirProvider boshdir.Provider
	logger      logger.Logger
	backupPath  string
}

const rootCertStore string = `Cert:\LocalMachine\Root`

func NewWindowsCertManager(fs boshsys.FileSystem, runner boshsys.CmdRunner, dirProvider boshdir.Provider, logger logger.Logger) Manager {
	return &windowsCertManager{
		fs:          fs,
		runner:      runner,
		dirProvider: dirProvider,
		logger:      logger,
		backupPath:  path.Join(dirProvider.TmpDir(), "rootCertBackup.sst"),
	}
}

func (c *windowsCertManager) createBackup() error {
	if _, err := os.Stat(c.backupPath); os.IsNotExist(err) {
		err = c.fs.MkdirAll(c.dirProvider.TmpDir(), os.FileMode(0777))
		if err != nil {
			return err
		}
		_, _, _, err := c.runner.RunCommand("powershell", "-Command",
			fmt.Sprintf(`"Get-ChildItem %s | Export-Certificate -Type SST -FilePath %s"`, rootCertStore, c.backupPath))
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *windowsCertManager) resetCerts() error {
	_, _, _, err := c.runner.RunCommand("powershell", "-Command", fmt.Sprintf(`Remove-Item %s\*`, rootCertStore))
	if err != nil {
		return err
	}

	importCertsCmd := fmt.Sprintf("Import-Certificate -FilePath %s -CertStoreLocation %s", c.backupPath, rootCertStore)
	_, _, _, err = c.runner.RunCommand("powershell", "-Command", importCertsCmd)
	if err != nil {
		return err
	}
	return nil
}

func (c *windowsCertManager) UpdateCertificates(rawCerts string) error {
	err := c.createBackup()
	if err != nil {
		return err
	}

	err = c.resetCerts()
	if err != nil {
		return err
	}

	certs := splitCerts(rawCerts)
	tempCertDir, err := c.fs.TempDir("")
	if err != nil {
		return err
	}
	defer c.fs.RemoveAll(tempCertDir)

	for i, cert := range certs {
		filename := path.Join(tempCertDir, strconv.Itoa(i))
		err = c.fs.WriteFileString(filename, cert)
		if err != nil {
			return err
		}
		_, _, _, err = c.runner.RunCommand("powershell", "-Command",
			fmt.Sprintf("Import-Certificate -FilePath %s -CertStoreLocation %s", filename, rootCertStore))
		if err != nil {
			return err
		}
	}
	return nil
}
