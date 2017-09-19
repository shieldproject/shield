package cmd_test

import (
	"errors"

	semver "github.com/cppforlife/go-semi-semantic/version"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	fakecmd "github.com/cloudfoundry/bosh-cli/cmd/cmdfakes"
	boshrel "github.com/cloudfoundry/bosh-cli/release"
	fakerel "github.com/cloudfoundry/bosh-cli/release/releasefakes"
)

var _ = Describe("ReleaseManager", func() {
	var (
		createReleaseCmd *fakecmd.FakeReleaseCreatingCmd
		uploadReleaseCmd *fakecmd.FakeReleaseUploadingCmd
		releaseManager   ReleaseManager
	)

	BeforeEach(func() {
		createReleaseCmd = &fakecmd.FakeReleaseCreatingCmd{
			RunStub: func(opts CreateReleaseOpts) (boshrel.Release, error) {
				release := &fakerel.FakeRelease{
					NameStub:    func() string { return opts.Name },
					VersionStub: func() string { return opts.Name + "-created-ver" },
				}
				return release, nil
			},
		}

		uploadReleaseCmd = &fakecmd.FakeReleaseUploadingCmd{}

		releaseManager = NewReleaseManager(createReleaseCmd, uploadReleaseCmd)
	})

	Describe("UploadReleases", func() {
		It("uploads remote releases skipping releases without url", func() {
			bytes := []byte(`
releases:
- name: capi
  sha1: capi-sha1
  url: https://capi-url
  version: 1+capi
- name: rel-without-upload
  version: 1+rel
- name: consul
  sha1: consul-sha1
  url: https://consul-url
  version: 1+consul
- name: local
  url: file:///local-dir
  version: create
`)

			_, err := releaseManager.UploadReleases(bytes)
			Expect(err).ToNot(HaveOccurred())

			Expect(uploadReleaseCmd.RunCallCount()).To(Equal(3))

			Expect(uploadReleaseCmd.RunArgsForCall(0)).To(Equal(UploadReleaseOpts{
				Name:    "capi",
				Args:    UploadReleaseArgs{URL: URLArg("https://capi-url")},
				SHA1:    "capi-sha1",
				Version: VersionArg(semver.MustNewVersionFromString("1+capi")),
			}))

			Expect(uploadReleaseCmd.RunArgsForCall(1)).To(Equal(UploadReleaseOpts{
				Name:    "consul",
				Args:    UploadReleaseArgs{URL: URLArg("https://consul-url")},
				SHA1:    "consul-sha1",
				Version: VersionArg(semver.MustNewVersionFromString("1+consul")),
			}))

			arg := uploadReleaseCmd.RunArgsForCall(2)
			Expect(arg.Release.Name()).To(Equal("local"))
			Expect(arg).To(Equal(UploadReleaseOpts{Release: arg.Release})) // only Release should be set
		})

		It("skips uploading releases if url is not provided, even if the version is invalid", func() {
			bytes := []byte(`
releases:
- name: capi
  version: ((/blah_interpolate_me_with_config_server))
`)

			_, err := releaseManager.UploadReleases(bytes)
			Expect(err).ToNot(HaveOccurred())
			Expect(uploadReleaseCmd.RunCallCount()).To(Equal(0))
		})

		It("creates releases if version is 'create' skipping others", func() {
			bytes := []byte(`
releases:
- name: capi
  url: file:///capi-dir
  version: create
- name: rel-without-upload
  version: 1+rel
- name: consul
  url: /consul-dir # doesn't require file://
  version: create
`)

			bytes, err := releaseManager.UploadReleases(bytes)
			Expect(err).ToNot(HaveOccurred())

			Expect(createReleaseCmd.RunCallCount()).To(Equal(2))

			Expect(createReleaseCmd.RunArgsForCall(0)).To(Equal(CreateReleaseOpts{
				Name:             "capi",
				Directory:        DirOrCWDArg{Path: "/capi-dir"},
				TimestampVersion: true,
				Force:            true,
			}))

			Expect(createReleaseCmd.RunArgsForCall(1)).To(Equal(CreateReleaseOpts{
				Name:             "consul",
				Directory:        DirOrCWDArg{Path: "/consul-dir"},
				TimestampVersion: true,
				Force:            true,
			}))

			Expect(bytes).To(Equal([]byte(`releases:
- name: capi
  version: capi-created-ver
- name: rel-without-upload
  version: 1+rel
- name: consul
  version: consul-created-ver
`)))
		})

		It("returns error and does not upload if creating release fails", func() {
			bytes := []byte(`
releases:
- name: capi
  url: file:///capi-dir
  version: create
`)
			createReleaseCmd.RunReturns(nil, errors.New("fake-err"))

			_, err := releaseManager.UploadReleases(bytes)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))

			Expect(uploadReleaseCmd.RunCallCount()).To(Equal(0))
		})

		It("returns error if uploading release fails", func() {
			bytes := []byte(`
releases:
- name: capi
  sha1: capi-sha1
  url: https://capi-url
  version: 1+capi
`)
			uploadReleaseCmd.RunReturns(errors.New("fake-err"))

			_, err := releaseManager.UploadReleases(bytes)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("returns an error and does not upload if release version cannot be parsed", func() {
			bytes := []byte(`
releases:
- name: capi
  sha1: capi-sha1
  url: https://capi-url
  version: 1+capi+capi
`)

			_, err := releaseManager.UploadReleases(bytes)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Expected version '1+capi+capi' to match version format"))

			Expect(uploadReleaseCmd.RunCallCount()).To(Equal(0))
		})

		It("returns an error if bytes cannot be parsed to find releases", func() {
			bytes := []byte(`-`)

			_, err := releaseManager.UploadReleases(bytes)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Parsing manifest"))

			Expect(createReleaseCmd.RunCallCount()).To(Equal(0))
			Expect(uploadReleaseCmd.RunCallCount()).To(Equal(0))
		})
	})
})
