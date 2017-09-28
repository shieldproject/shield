package config_test

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/starkandwayne/shield/api"
	. "github.com/starkandwayne/shield/cmd/shield/config"
)

var _ = Describe("Config dirtiness", func() {
	//Dirty checks are mostly an internal thing, which I don't want to worry about
	//testing too much. This set of tests is concerned with finding
	//false-negatives - that is, that the config doesn't write when it should
	//have.

	BeforeEach(func() {
		Initialize()
	})

	var testPath string
	BeforeEach(func() {
		tempFile, err := ioutil.TempFile("", "shield-test-cfg")
		Expect(err).NotTo(HaveOccurred(), "Could not create file in temp dir")
		testPath = tempFile.Name()
		Expect(tempFile.Close()).To(Succeed(), "Could not close temp file")
		//Load an empty config to give Save a target to save to. Innocent hack that
		//will work for as long as Load()ing an empty file actually produces an empty
		//config.
		Expect(Load(testPath)).To(Succeed(), "Could not load empty config")
		Expect(List()).To(BeEmpty(), "Loading empty config didn't make empty config")
	})

	AfterEach(func() {
		Expect(os.Remove(testPath)).To(Succeed(), "Could not remove test file")
	})

	var cycleConfig = func() {
		Expect(Save()).To(Succeed(), "Could not save to temp file")
		Expect(Load(testPath)).To(Succeed(), "Could not load from temp file")
	}

	var testChangeSticks = func() {
		var before, after []*api.Backend

		JustBeforeEach(func() {
			before = List()
			cycleConfig()
			after = List()
		})

		Specify("The change should be accessible after loading", func() {
			Expect(after).To(Equal(before))
		})
	}

	Context("When a commit was made", func() {
		var toCommit *api.Backend

		JustBeforeEach(func() {
			Expect(Commit(toCommit)).To(Succeed())
		})

		AfterEach(func() {
			toCommit = nil
		})

		Context("When a new thing is committed", func() {
			BeforeEach(func() {
				toCommit = &api.Backend{
					Name:              "thing",
					Address:           "http://thing",
					Token:             "basic thing",
					SkipSSLValidation: true,
					CACert: `-----BEGIN CERTIFICATE---
-----END CERTIFICATE-----`,
				}
			})

			testChangeSticks()
		})

		Context("When something with the same name as an existing backend is committed", func() {
			var original *api.Backend

			BeforeEach(func() {
				original = &api.Backend{
					Name:              "thing",
					Address:           "http://thing",
					Token:             "basic thing",
					SkipSSLValidation: true,
					CACert: `-----BEGIN CERTIFICATE---
-----END CERTIFICATE-----`,
				}
				Expect(Commit(original)).To(Succeed())
				cycleConfig()
			})

			AfterEach(func() {
				original = nil
			})

			Context("But with a different address", func() {
				BeforeEach(func() {
					updated := *original
					updated.Address = "http://newthing"
					toCommit = &updated
				})

				testChangeSticks()
			})

			Context("But with a different token", func() {
				BeforeEach(func() {
					updated := *original
					updated.Token = "basic newthing"
					toCommit = &updated
				})

				testChangeSticks()
			})

			Context("But with skip SSL validation toggled off", func() {
				BeforeEach(func() {
					updated := *original
					updated.SkipSSLValidation = false
					toCommit = &updated
				})

				testChangeSticks()
			})
		})
	})

	Context("When a new backend is selected for use", func() {
		var toUse string
		JustBeforeEach(func() {
			Expect(Use(toUse)).To(Succeed())
		})

		Context("and no backend was previously selected", func() {
			BeforeEach(func() {
				original := &api.Backend{
					Name:    "thing",
					Address: "http://thing",
					Token:   "basic thing",
					CACert: `-----BEGIN CERTIFICATE---
-----END CERTIFICATE-----`,
				}
				Expect(Commit(original)).To(Succeed())
				toUse = original.Name
			})

			testChangeSticks()
		})

		Context("and there was already a backend selected", func() {
			BeforeEach(func() {
				first := &api.Backend{
					Name:    "thing",
					Address: "http://thing",
					Token:   "basic thing",
					CACert: `-----BEGIN CERTIFICATE---
-----END CERTIFICATE-----`,
				}
				Expect(Commit(first)).To(Succeed())
				second := &api.Backend{
					Name:    "thing2",
					Address: "http://thing2",
					Token:   "basic thing2",
					CACert: `-----BEGIN CERTIFICATE---
-----END CERTIFICATE-----`,
				}
				Expect(Commit(second)).To(Succeed())

				Expect(Use(first.Name)).To(Succeed())
				cycleConfig()

				toUse = second.Name
			})

			testChangeSticks()
		})
	})

	Context("When a backend is deleted", func() {
		var toDelete string
		JustBeforeEach(func() {
			Expect(Delete(toDelete)).To(Succeed())
		})

		BeforeEach(func() {
			inserted := &api.Backend{
				Name:    "thing",
				Address: "http://thing",
				Token:   "basic token",
				CACert: `-----BEGIN CERTIFICATE---
-----END CERTIFICATE-----`,
			}
			Expect(Commit(inserted)).To(Succeed())
			cycleConfig()
			toDelete = inserted.Name
		})

		testChangeSticks()
	})
})
