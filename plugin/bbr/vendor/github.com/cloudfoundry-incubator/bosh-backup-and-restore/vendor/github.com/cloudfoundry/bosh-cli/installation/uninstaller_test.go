package installation_test

import (
	"errors"
	"log"
	"path/filepath"
	"regexp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/bosh-cli/installation"
	"github.com/cloudfoundry/bosh-utils/logger"
	"github.com/cloudfoundry/bosh-utils/system"
	"github.com/cloudfoundry/bosh-utils/system/fakes"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Uninstaller", func() {
	Describe("Uninstall", func() {
		It("deletes the installation target directory", func() {
			logBuffer := gbytes.NewBuffer()
			goLogger := log.New(logBuffer, "", log.LstdFlags)
			boshlogger := logger.New(logger.LevelInfo, goLogger, goLogger)

			fs := system.NewOsFileSystem(boshlogger)
			installationPath, err := fs.TempDir("some-installation-dir")
			Expect(err).ToNot(HaveOccurred())

			err = fs.WriteFileString(filepath.Join(installationPath, "some-installation-artifact"), "something-blah")
			Expect(err).ToNot(HaveOccurred())

			installationTarget := installation.NewTarget(installationPath)

			uninstaller := installation.NewUninstaller(fs, boshlogger)

			Expect(fs.FileExists(installationPath)).To(BeTrue())

			err = uninstaller.Uninstall(installationTarget)
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.FileExists(installationPath)).To(BeFalse())
			Expect(logBuffer).To(gbytes.Say("Successfully uninstalled CPI from '%s'", regexp.QuoteMeta(installationPath)))
		})

		It("returns and logs errors when remove all fails", func() {
			logBuffer := gbytes.NewBuffer()
			goLogger := log.New(logBuffer, "", log.LstdFlags)
			boshlogger := logger.New(logger.LevelInfo, goLogger, goLogger)

			fs := fakes.NewFakeFileSystem()
			fs.RemoveAllStub = func(_ string) error {
				return errors.New("can't remove that")
			}

			installationTarget := installation.NewTarget("/not/a/path")

			uninstaller := installation.NewUninstaller(fs, boshlogger)

			err := uninstaller.Uninstall(installationTarget)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("can't remove that"))

			Expect(logBuffer).To(gbytes.Say("Failed to uninstall CPI from '/not/a/path': can't remove that"))
		})
	})
})
