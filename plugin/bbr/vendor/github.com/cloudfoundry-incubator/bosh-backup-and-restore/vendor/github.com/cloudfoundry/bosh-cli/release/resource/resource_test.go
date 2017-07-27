package resource_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/bosh-cli/crypto/fakes"
	. "github.com/cloudfoundry/bosh-cli/release/resource"
	fakeres "github.com/cloudfoundry/bosh-cli/release/resource/resourcefakes"
	boshcrypto "github.com/cloudfoundry/bosh-utils/crypto"
	fakesfs "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("NewResource", func() {
	var (
		devIndex, finalIndex *fakeres.FakeArchiveIndex
		archive              *fakeres.FakeArchive
		resource             Resource
	)

	BeforeEach(func() {
		devIndex = &fakeres.FakeArchiveIndex{}
		finalIndex = &fakeres.FakeArchiveIndex{}
		archive = &fakeres.FakeArchive{}
		resource = NewResource("name", "fp", archive)
	})

	Describe("common methods", func() {
		It("returns name", func() {
			Expect(resource.Name()).To(Equal("name"))
		})
	})

	Describe("Fingerprint", func() {
		It("returns fp", func() {
			Expect(resource.Fingerprint()).To(Equal("fp"))
		})
	})

	Describe("ArchivePath", func() {
		It("panics before building", func() {
			Expect(func() { resource.ArchivePath() }).To(Panic())
		})
	})

	Describe("ArchiveSHA1", func() {
		It("panics before building", func() {
			Expect(func() { resource.ArchiveSHA1() }).To(Panic())
		})
	})

	Describe("Build", func() {
		It("associated resource with found archive from dev index", func() {
			devIndex.FindStub = func(name, fp string) (string, string, error) {
				return "/found", "sha1", nil
			}

			Expect(resource.Build(devIndex, finalIndex)).ToNot(HaveOccurred())

			Expect(resource.ArchivePath()).To(Equal("/found"))
			Expect(resource.ArchiveSHA1()).To(Equal("sha1"))
		})

		It("returns error when dev index check fails", func() {
			devIndex.FindStub = func(name, fp string) (string, string, error) {
				return "", "", errors.New("fake-err")
			}

			err := resource.Build(devIndex, finalIndex)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("associated resource with found archive from final index", func() {
			finalIndex.FindStub = func(name, fp string) (string, string, error) {
				return "/found", "sha1", nil
			}

			Expect(resource.Build(devIndex, finalIndex)).ToNot(HaveOccurred())

			Expect(resource.ArchivePath()).To(Equal("/found"))
			Expect(resource.ArchiveSHA1()).To(Equal("sha1"))
		})

		It("returns error when final index check fails", func() {
			finalIndex.FindStub = func(name, fp string) (string, string, error) {
				return "", "", errors.New("fake-err")
			}

			err := resource.Build(devIndex, finalIndex)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("builds archive and adds to dev index when dev or final indicies don't have it", func() {
			archive.BuildReturns("/built", "built-sha1", nil)

			Expect(resource.Build(devIndex, finalIndex)).ToNot(HaveOccurred())

			Expect(devIndex.AddCallCount()).To(Equal(1))

			name, fp, path, sha1 := devIndex.AddArgsForCall(0)
			Expect(name).To(Equal("name"))
			Expect(fp).To(Equal("fp"))
			Expect(path).To(Equal("/built"))
			Expect(sha1).To(Equal("built-sha1"))
		})

		It("returns error if archive building fails", func() {
			archive.BuildReturns("", "", errors.New("fake-err"))

			err := resource.Build(devIndex, finalIndex)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("returns error when dev index addition fails of newly built archive", func() {
			archive.BuildReturns("/built", "built-sha1", nil)

			devIndex.AddReturns("", "", errors.New("fake-err"))

			err := resource.Build(devIndex, finalIndex)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})

	Describe("Finalize", func() {
		It("uses existing resource asset (path/sha1) when there is already finalized resource", func() {
			resource = NewResourceWithBuiltArchive("name", "fp", "/prev-path", "prev-sha1")

			finalIndex.FindStub = func(name, fp string) (string, string, error) {
				Expect(name).To(Equal("name"))
				Expect(fp).To(Equal("fp"))
				return "/found", "found-sha1", nil
			}

			Expect(resource.Finalize(finalIndex)).ToNot(HaveOccurred())
			Expect(finalIndex.AddCallCount()).To(Equal(0))
			Expect(resource.ArchivePath()).To(Equal("/found"))
			Expect(resource.ArchiveSHA1()).To(Equal("found-sha1"))
		})

		It("returns error when final index check fails", func() {
			finalIndex.FindStub = func(name, fp string) (string, string, error) {
				return "/found", "", errors.New("fake-err")
			}

			err := resource.Finalize(finalIndex)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("panics without building", func() {
			Expect(func() { resource.Finalize(finalIndex) }).To(Panic())
		})

		buildBeforeFinalizing := func() {
			devIndex.FindStub = func(name, fp string) (string, string, error) {
				return "/found", "found-sha1", nil
			}

			Expect(resource.Build(devIndex, finalIndex)).ToNot(HaveOccurred())
		}

		It("adds archive to final index when final index does not already have archive", func() {
			buildBeforeFinalizing()

			Expect(resource.Finalize(finalIndex)).ToNot(HaveOccurred())

			Expect(finalIndex.AddCallCount()).To(Equal(1))

			name, fp, path, sha1 := finalIndex.AddArgsForCall(0)
			Expect(name).To(Equal("name"))
			Expect(fp).To(Equal("fp"))
			Expect(path).To(Equal("/found"))
			Expect(sha1).To(Equal("found-sha1"))
		})

		It("returns error when final index addition fails", func() {
			buildBeforeFinalizing()

			finalIndex.AddReturns("", "", errors.New("fake-err"))

			err := resource.Finalize(finalIndex)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})

var _ = Describe("NewExistingResource", func() {
	var (
		devIndex, finalIndex *fakeres.FakeArchiveIndex
		resource             Resource
	)

	BeforeEach(func() {
		devIndex = &fakeres.FakeArchiveIndex{}
		finalIndex = &fakeres.FakeArchiveIndex{}
		resource = NewExistingResource("name", "fp", "sha1")
	})

	Describe("Name", func() {
		It("returns name", func() {
			Expect(resource.Name()).To(Equal("name"))
		})
	})

	Describe("Fingerprint", func() {
		It("returns fp", func() {
			Expect(resource.Fingerprint()).To(Equal("fp"))
		})
	})

	Describe("ArchivePath", func() {
		It("panics before building", func() {
			Expect(func() { resource.ArchivePath() }).To(Panic())
		})
	})

	Describe("ArchiveSHA1", func() {
		It("returns sha1", func() {
			Expect(resource.ArchiveSHA1()).To(Equal("sha1"))
		})
	})

	Describe("Build", func() {
		It("associated resource with found archive from dev index", func() {
			devIndex.FindStub = func(name, fp string) (string, string, error) {
				return "/found", "found-sha1", nil
			}

			err := resource.Build(devIndex, finalIndex)
			Expect(err).ToNot(HaveOccurred())

			Expect(resource.ArchivePath()).To(Equal("/found"))
			Expect(resource.ArchiveSHA1()).To(Equal("found-sha1"))
		})

		It("returns error when dev index check fails", func() {
			devIndex.FindStub = func(name, fp string) (string, string, error) {
				return "", "", errors.New("fake-err")
			}

			err := resource.Build(devIndex, finalIndex)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("associated resource with found archive from final index", func() {
			finalIndex.FindStub = func(name, fp string) (string, string, error) {
				return "/found", "found-sha1", nil
			}

			err := resource.Build(devIndex, finalIndex)
			Expect(err).ToNot(HaveOccurred())

			Expect(resource.ArchivePath()).To(Equal("/found"))
			Expect(resource.ArchiveSHA1()).To(Equal("found-sha1"))
		})

		It("returns error when final index check fails", func() {
			finalIndex.FindStub = func(name, fp string) (string, string, error) {
				return "", "", errors.New("fake-err")
			}

			err := resource.Build(devIndex, finalIndex)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("returns error when dev or final indicies don't have it", func() {
			err := resource.Build(devIndex, finalIndex)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected to find 'name/fp'"))
		})
	})

	Describe("Finalize", func() {
		It("does nothing when there is already finalized resource", func() {
			finalIndex.FindStub = func(name, fp string) (string, string, error) {
				Expect(name).To(Equal("name"))
				Expect(fp).To(Equal("fp"))
				return "/found", "", nil
			}

			Expect(resource.Finalize(finalIndex)).ToNot(HaveOccurred())
			Expect(finalIndex.AddCallCount()).To(Equal(0))
		})

		It("returns error when final index check fails", func() {
			finalIndex.FindStub = func(name, fp string) (string, string, error) {
				return "/found", "", errors.New("fake-err")
			}

			err := resource.Finalize(finalIndex)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("panics without building", func() {
			Expect(func() { resource.Finalize(finalIndex) }).To(Panic())
		})

		buildBeforeFinalizing := func() {
			devIndex.FindStub = func(name, fp string) (string, string, error) {
				return "/found", "found-sha1", nil
			}

			Expect(resource.Build(devIndex, finalIndex)).ToNot(HaveOccurred())
		}

		It("adds archive to final index when final index does not already have archive", func() {
			buildBeforeFinalizing()

			Expect(resource.Finalize(finalIndex)).ToNot(HaveOccurred())

			Expect(finalIndex.AddCallCount()).To(Equal(1))

			name, fp, path, sha1 := finalIndex.AddArgsForCall(0)
			Expect(name).To(Equal("name"))
			Expect(fp).To(Equal("fp"))
			Expect(path).To(Equal("/found"))
			Expect(sha1).To(Equal("found-sha1"))
		})

		It("returns error when final index addition fails", func() {
			buildBeforeFinalizing()

			finalIndex.AddReturns("", "", errors.New("fake-err"))

			err := resource.Finalize(finalIndex)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})

var _ = Describe("NewResourceWithBuiltArchive", func() {
	var (
		devIndex, finalIndex *fakeres.FakeArchiveIndex
		resource             Resource
		filePathName         string
		filePathNameSha1     string
		fakeFs               *fakesfs.FakeFileSystem
	)

	BeforeEach(func() {
		devIndex = &fakeres.FakeArchiveIndex{}
		finalIndex = &fakeres.FakeArchiveIndex{}

		fakeFs = fakesfs.NewFakeFileSystem()
		file, err := fakeFs.TempFile("path")
		Expect(err).ToNot(HaveOccurred())

		fakeFs.RegisterOpenFile(file.Name(), &fakesfs.FakeFile{
			Contents: []byte("hello world"),
		})

		filePathName = file.Name()
		filePathNameSha1 = "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed"
		resource = NewResourceWithBuiltArchive("name", "fp", filePathName, filePathNameSha1)
	})

	Describe("Name", func() {
		It("returns name", func() {
			Expect(resource.Name()).To(Equal("name"))
		})
	})

	Describe("Fingerprint", func() {
		It("returns fp", func() {
			Expect(resource.Fingerprint()).To(Equal("fp"))
		})
	})

	Describe("ArchivePath", func() {
		It("returns path", func() {
			Expect(resource.ArchivePath()).To(Equal(filePathName))
		})
	})

	Describe("ArchiveSHA1", func() {
		It("returns sha1", func() {
			Expect(resource.ArchiveSHA1()).To(Equal(filePathNameSha1))
		})
	})

	Describe("RehashWithCalculator", func() {
		Context("Given a sha256 calculator", func() {
			var fakeDigestCalculator *fakes.FakeDigestCalculator

			BeforeEach(func() {
				fakeDigestCalculator = fakes.NewFakeDigestCalculator()
				fakeDigestCalculator.SetCalculateBehavior(map[string]fakes.CalculateInput{
					filePathName: {DigestStr: "sha256:new_resource_sha"},
				})
			})

			Context("Given a resource with a valid sha128", func() {
				It("A copy of a resource with sha256", func() {
					newSha256Resource, err := resource.RehashWithCalculator(fakeDigestCalculator, boshcrypto.ArchiveDigestFilePathReader(fakeFs))
					Expect(err).ToNot(HaveOccurred())

					Expect(newSha256Resource.ArchiveSHA1()).To(Equal("sha256:new_resource_sha"))
				})
			})

			Context("Given a resource with an invalid sha128", func() {
				BeforeEach(func() {
					resource = NewResourceWithBuiltArchive("name", "fp", filePathName, "bad")
				})

				It("an error should occur", func() {
					_, err := resource.RehashWithCalculator(fakeDigestCalculator, boshcrypto.ArchiveDigestFilePathReader(fakeFs))
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Expected stream to have digest"))
				})
			})

		})

	})

	Describe("Build", func() {
		It("does nothing because we already have archive", func() {
			Expect(resource.Build(devIndex, finalIndex)).ToNot(HaveOccurred())
		})
	})

	Describe("Finalize", func() {
		It("does nothing when there is already finalizes resource", func() {
			finalIndex.FindStub = func(name, fp string) (string, string, error) {
				Expect(name).To(Equal("name"))
				Expect(fp).To(Equal("fp"))
				return "/found", "", nil
			}

			Expect(resource.Finalize(finalIndex)).ToNot(HaveOccurred())
			Expect(finalIndex.AddCallCount()).To(Equal(0))
		})

		It("returns error when final index check fails", func() {
			finalIndex.FindStub = func(name, fp string) (string, string, error) {
				return "/found", "", errors.New("fake-err")
			}

			err := resource.Finalize(finalIndex)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("adds archive to final index when final index does not already have archive", func() {
			Expect(resource.Finalize(finalIndex)).ToNot(HaveOccurred())

			Expect(finalIndex.AddCallCount()).To(Equal(1))

			name, fp, path, sha1 := finalIndex.AddArgsForCall(0)
			Expect(name).To(Equal("name"))
			Expect(fp).To(Equal("fp"))
			Expect(path).To(Equal(filePathName))
			Expect(sha1).To(Equal(filePathNameSha1))
		})

		It("returns error when final index addition fails", func() {
			finalIndex.AddReturns("", "", errors.New("fake-err"))

			err := resource.Finalize(finalIndex)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
