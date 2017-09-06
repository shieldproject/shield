package config_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/starkandwayne/shield/api"
	. "github.com/starkandwayne/shield/cmd/shield/config"
)

var _ = Describe("When loading configs", func() {
	var testPath string

	AfterEach(func() {
		testPath = ""
	})

	JustBeforeEach(func() {
		err = Load(testPath)
	})

	Context("When the config does not have read permissions", func() {
		BeforeEach(func() {
			if os.Geteuid() == 0 {
				Skip("Cannot test unreadable files when euid = 0")
			}

			testPath = "test/etc/valid.yml"
			os.Chmod(testPath, 0200)
		})

		AfterEach(func() {
			os.Chmod(testPath, 0644)
		})

		It("throws an error", func() {
			Expect(err).To(HaveOccurred())
		})
	})

	Context("When the yaml in the config is invalid", func() {
		BeforeEach(func() {
			testPath = "test/etc/invalid.yml"
		})
		It("throws an error", func() {
			Expect(err).To(HaveOccurred())
		})
	})

	Context("The config is not present", func() {
		BeforeEach(func() {
			testPath = "test/etc/missing.yml"
		})

		It("should not err", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should initialize with an empty config", func() {
			Expect(List()).To(BeEmpty())
		})

		It("should be prepared to write to the configured path", func() {
			Expect(Path()).To(Equal(testPath))
		})
	})

	Context("When the config is valid, present, and non-empty", func() {
		BeforeEach(func() {
			testPath = "test/etc/valid.yml"
		})

		It("should not err", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should be prepared to write to the configured path", func() {
			Expect(Path()).To(Equal(testPath))
		})

		It("should load the correct number of backends", func() {
			Expect(List()).To(HaveLen(2))
		})

		Specify("the first backend should have the expected information", func() {
			Expect(*Get("first")).To(Equal(api.Backend{
				Name:    "first",
				Address: "http://first",
				Token:   "basic mytoken1",
			}))
		})

		Specify("the second backend should have the expected information", func() {
			Expect(*Get("second")).To(Equal(api.Backend{
				Name:              "second",
				Address:           "http://second",
				SkipSSLValidation: true,
			}))
		})
	})
})
