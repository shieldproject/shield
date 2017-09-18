package config_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/starkandwayne/shield/cmd/shield/config"
	yaml "gopkg.in/yaml.v2"
)

var _ = Describe("When saving configs", func() {
	var testPath string
	var shouldDirtyConfig bool

	BeforeEach(func() {
		tempFile, err := ioutil.TempFile("", "shield-test-cfg")
		Expect(err).NotTo(HaveOccurred(), "Could not create file in temp dir")
		testPath = tempFile.Name()
		//Copy our expected config to the temp location, creating a version that
		// we don't care if Save screws up
		var contents []byte
		contents, err = ioutil.ReadFile("test/etc/valid.yml")
		Expect(err).NotTo(HaveOccurred(), "Could not read from valid.yml in order to test Save()")
		n, err := tempFile.Write(contents)
		Expect(err).NotTo(HaveOccurred(), "An error occurred when copying the test file in prep to test Save()")
		Expect(n).To(Equal(len(contents)), "Could not copy all of the contents of the test file in prep to test Save()")
		Expect(tempFile.Close()).To(Succeed(), "An error was reported when closing the temp file in prep to test Save()")
	})

	AfterEach(func() {
		if _, err = os.Stat(testPath); err != nil {
			if !os.IsNotExist(err) {
				Warn("Could not stat temp file at `%s': %s", testPath, err)
			}
		} else {
			err = os.Remove(testPath)
			if err != nil {
				Warn("Could not remove temp file at `%s': %s", testPath, err)
			}
		}

		testPath = ""
		shouldDirtyConfig = false
	})

	JustBeforeEach(func() {
		Expect(Load(testPath)).To(Succeed(), "Could not load from config in order to test Save")
		if shouldDirtyConfig {
			//Make config dirty in order to force saves to actually save
			Expect(Delete("second")).To(Succeed(), "Could not delete object to force a save")
		}
		err = Save()
	})

	wasSaved := func() bool {
		var contents []byte
		contents, err = ioutil.ReadFile(testPath)
		Expect(err).NotTo(HaveOccurred(), "Could not read file at test path")
		conf := map[string]interface{}{}
		Expect(yaml.Unmarshal(contents, &conf)).To(Succeed(), "Unable to unmarshal yaml from file after it was saved")
		return conf["not_saved"] == nil
	}

	Context("When the file location cannot be saved to", func() {
		var tempDir string
		BeforeEach(func() {
			if os.Geteuid() == 0 {
				Skip("Cannot test unwritable file location when euid = 0. Too much power")
			}

			tempDir, err = ioutil.TempDir("", "shield-config-test")

			newPath := fmt.Sprintf("%s/%s", tempDir, filepath.Base(testPath))
			Expect(os.Rename(testPath, newPath)).To(Succeed(), "Could not move file into temp dir")
			testPath = newPath
			Expect(os.Chmod(tempDir, 0500)).To(Succeed(), "Could not reduce file permissions")
			shouldDirtyConfig = true
		})

		AfterEach(func() {
			if err = os.Chmod(tempDir, 0755); err != nil {
				Warn("Could not restore permissions to tmp dir `%s': %s", tempDir, err)
			}

			if err = os.RemoveAll(tempDir); err != nil {
				Warn("Could not remove temp directory `%s': %s", tempDir, err)
			}
			tempDir = ""
		})

		It("Throws an error", func() {
			Expect(err).To(HaveOccurred())
		})
	})

	Context("When the file should be saveable", func() {

		AfterEach(func() {
		})

		Context("when the config is dirty", func() {
			BeforeEach(func() {
				shouldDirtyConfig = true
			})

			It("should not err", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("successfully writes the config to disk", func() {
				Expect(wasSaved()).To(BeTrue())
			})
		})

		Context("when the config is not dirty", func() {
			It("should not err", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should not write to disk", func() {
				Expect(wasSaved()).To(BeFalse())
			})
		})
	})
})
