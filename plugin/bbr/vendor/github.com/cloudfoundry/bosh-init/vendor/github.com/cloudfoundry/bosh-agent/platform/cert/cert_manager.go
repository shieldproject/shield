package cert

import (
	"fmt"
	"strings"
	"time"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	"github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

//go:generate counterfeiter . Manager

// Manager is a set of operations for manipulating the set of trusted CA certificates
// on any OS platform.
type Manager interface {

	// UpdateCertificates manages the set of CA certificates that are trusted in
	// addition to certificates that were pre-installed on the operating system.
	//
	// Each call alters the set of X.509 certificates that are trusted as
	// root certificates on this machine to match the set of certificates given.
	//
	// Calling this method again later with a different set of certificates will
	// replace the previously trusted certificates with the new set; hence, calling
	// this method with an empty set of certificates will bring this machine back to
	// the initial state, where it only trusts the CA certificates that came with the OS.
	//
	// The certs argument should contain zero or more X.509 certificates in PEM format
	// concatenated together. Any text that is not between `-----BEGIN CERTIFICATE-----`
	// and `-----END CERTIFICATE-----` lines is ignored.
	UpdateCertificates(certs string) error
}

type certManager struct {
	fs            boshsys.FileSystem
	runner        boshsys.CmdRunner
	path          string
	updateCmdPath string
	updateCmdArgs []string
	logger        logger.Logger
	logTag        string
	// Update execution time limit in seconds
	// No retry if 0
	updateTimeout time.Duration
}

func NewUbuntuCertManager(fs boshsys.FileSystem, runner boshsys.CmdRunner, timeout time.Duration, logger logger.Logger) Manager {
	return &certManager{
		fs:            fs,
		runner:        runner,
		path:          "/usr/local/share/ca-certificates/",
		updateCmdPath: "/usr/sbin/update-ca-certificates",
		updateCmdArgs: []string{"-f"},
		logger:        logger,
		logTag:        "UbuntuCertManager",
		updateTimeout: timeout,
	}
}

func NewCentOSCertManager(fs boshsys.FileSystem, runner boshsys.CmdRunner, timeout time.Duration, logger logger.Logger) Manager {
	return &certManager{
		fs:            fs,
		runner:        runner,
		path:          "/etc/pki/ca-trust/source/anchors/",
		updateCmdPath: "/usr/bin/update-ca-trust",
		logger:        logger,
		logTag:        "CentOSCertManager",
		updateTimeout: timeout,
	}
}

func NewDummyCertManager(fs boshsys.FileSystem, runner boshsys.CmdRunner, timeout time.Duration, logger logger.Logger) Manager {
	return &certManager{
		fs:            fs,
		runner:        runner,
		path:          "dummy",
		updateCmdPath: "dummy",
		logger:        logger,
		logTag:        "DummyCertManager",
		updateTimeout: timeout,
	}
}

func (c *certManager) UpdateCertificates(certs string) error {
	c.logger.Info(c.logTag, "Running Update Certificate command")

	if c.updateCmdPath == "dummy" {
		return nil
	}

	deletedFilesCount, err := deleteFiles(c.fs, c.path, "bosh-trusted-cert-")
	c.logger.Debug(c.logTag, "Deleted %d existing certificate files", deletedFilesCount)
	if err != nil {
		return err
	}

	slicedCerts := splitCerts(certs)
	for i, cert := range slicedCerts {
		err := c.fs.WriteFileString(fmt.Sprintf("%sbosh-trusted-cert-%d.crt", c.path, i+1), cert)
		if err != nil {
			return err
		}
	}
	c.logger.Debug(c.logTag, "Wrote %d new certificate files", len(slicedCerts))

	// For Ubuntu OS, update-ca-certificates occasionally hangs, which results
	// in bosh-agent failure. A retry normally solves this issue. We kill the process
	// if it runs over given time limit and retry for 3 times until we throw error.
	if c.updateTimeout > 0 {
		command := boshsys.Command{
			Name: c.updateCmdPath,
			Args: c.updateCmdArgs,
		}

		for i := 1; i < 4; i++ {
			c.logger.Debug(c.logTag, "Try to update new certificate files with retry, take %d of 3", i)

			process, err := c.runner.RunComplexCommandAsync(command)
			if err != nil {
				return bosherr.WrapError(err, "Running command to update certificates with retries")
			}

			resultChannel := process.Wait()

			select {
			case <-time.After(c.updateTimeout * time.Second):
				err = process.TerminateNicely(5 * time.Second)
				if err != nil {
					c.logger.Debug(c.logTag, "Failed to terminate update certificates cmd '%s' after %d seconds", c.updateCmdPath, c.updateTimeout)
				}
			case result := <-resultChannel:
				if result.Error == nil {
					c.logger.Debug(c.logTag, "Successfully updated new certificate files")
					return nil
				}
			}
		}

		return bosherr.Error("Updating certificates with retries")
	}

	c.logger.Debug(c.logTag, "Try to update new certificate files without retry")

	_, _, _, err = c.runner.RunCommand(c.updateCmdPath, c.updateCmdArgs...)
	if err != nil {
		return bosherr.WrapError(err, "Running command to update certificates without retries")
	}

	c.logger.Debug(c.logTag, "Successfully updated new certificate files.")
	return nil
}

// SplitCerts returns a slice containing each PEM certificate in the given string.
// extra data before the first cert, between each cert, and after the last cert
// is all discarded. Each string in the returned slice will begin with
// `-----BEGIN CERTIFICATE-----` and end with `-----END CERTIFICATE-----`
// and have no leading or trailing whitespace.
func splitCerts(certs string) []string {
	result := strings.SplitAfter(fmt.Sprintln(certs), "-----END CERTIFICATE-----")
	for i := range result {
		start := strings.Index(result[i], "-----BEGIN CERTIFICATE-----")
		if start > 0 {
			result[i] = result[i][start:len(result[i])]
		}
	}
	return result[0 : len(result)-1]
}

func deleteFiles(fs boshsys.FileSystem, path string, filenamePrefix string) (int, error) {
	var deletedFilesCount int
	files, err := fs.Glob(fmt.Sprintf("%s%s*", path, filenamePrefix))
	if err != nil {
		return deletedFilesCount, bosherr.WrapError(err, "Glob command failed")
	}
	for _, file := range files {
		err = fs.RemoveAll(file)
		if err != nil {
			return deletedFilesCount, bosherr.WrapErrorf(err, "deleting %s failed", file)
		}
		deletedFilesCount++
	}
	return deletedFilesCount, err
}
