package backup_test

import (
	"fmt"
	"os"

	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/backup"
)

var _ = Context("BackupManager", func() {
	var deploymentName string
	var backupName string
	var backupManager = BackupDirectoryManager{}
	var err error
	fakeClock := func() time.Time {
		return time.Date(2015, 10, 21, 02, 2, 3, 0, time.FixedZone("UTC+1", 3600))
	}

	BeforeEach(func() {
		deploymentName = fmt.Sprintf("my-cool-redis-%d", config.GinkgoConfig.ParallelNode)
		backupName = deploymentName + "_20151021T010203Z"
	})

	AfterEach(func() {
		Expect(os.RemoveAll(backupName)).To(Succeed())
	})

	Describe("Create", func() {
		JustBeforeEach(func() {
			_, err = backupManager.Create(deploymentName, nil, fakeClock)
		})

		Context("when the directory exists", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(backupName, 0777)).To(Succeed())
			})

			It("returns an error", func() {
				Expect(err).To(MatchError(ContainSubstring("failed creating directory")))
			})
		})

		Context("when the directory doesnt exist", func() {
			It("creates a directory with the given name", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(backupName).To(BeADirectory())
			})
		})
	})

	Describe("Open", func() {
		Context("when the directory exists", func() {
			BeforeEach(func() {
				err := os.MkdirAll(backupName, 0700)
				Expect(err).NotTo(HaveOccurred())
			})

			It("does not create a directory", func() {
				_, err := backupManager.Open(backupName, nil)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the directory does not exist", func() {
			It("fails", func() {
				_, err := backupManager.Open(backupName, nil)
				Expect(err).To(MatchError(ContainSubstring("failed opening the directory")))
				Expect(backupName).NotTo(BeADirectory())
			})
		})
	})

	AfterEach(func() {
		Expect(os.RemoveAll(backupName)).To(Succeed())
	})
})
