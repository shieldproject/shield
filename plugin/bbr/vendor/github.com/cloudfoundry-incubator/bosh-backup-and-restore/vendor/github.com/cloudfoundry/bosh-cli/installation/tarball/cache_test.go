package tarball_test

import (
	"os"
	"path/filepath"
	"syscall"

	. "github.com/cloudfoundry/bosh-cli/installation/tarball"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cache", func() {
	var (
		cache Cache
		fs    *fakesys.FakeFileSystem
	)

	BeforeEach(func() {
		logger := boshlog.NewLogger(boshlog.LevelNone)
		fs = fakesys.NewFakeFileSystem()
		cache = NewCache(
			"/fake-base-path",
			fs,
			logger,
		)
	})

	It("is a cache hit when the tarball with that url and sha1 has been downloaded", func() {
		fs.WriteFileString("source-path", "")

		err := cache.Save("source-path", &fakeSource{
			sha1:        "fake-sha1",
			url:         "http://foo.bar.com",
			description: "some tarball",
		})
		Expect(err).ToNot(HaveOccurred())

		path, found := cache.Get(&fakeSource{
			sha1:        "fake-sha1",
			url:         "http://foo.bar.com",
			description: "some tarball",
		})
		Expect(found).To(BeTrue())
		Expect(fs.FileExists(path)).To(BeTrue())
	})

	It("is a cache miss when a tarball from a different url has been downloaded, even if SHA1 matches", func() {
		fs.WriteFileString("source-path", "")

		err := cache.Save("source-path", &fakeSource{
			sha1:        "fake-sha1",
			url:         "http://foo.bar.com",
			description: "some tarball",
		})
		Expect(err).ToNot(HaveOccurred())

		_, found := cache.Get(&fakeSource{
			sha1:        "fake-sha1",
			url:         "http://baz.bar.com",
			description: "some tarball",
		})
		Expect(found).To(BeFalse())
	})

	It("is a cache miss when a tarball from a different SHA1 has been downloaded, even if url matches", func() {
		fs.WriteFileString("source-path", "")

		err := cache.Save("source-path", &fakeSource{
			sha1:        "fake-sha1",
			url:         "http://foo.bar.com",
			description: "some tarball",
		})
		Expect(err).ToNot(HaveOccurred())

		_, found := cache.Get(&fakeSource{
			sha1:        "different-fake-sha1",
			url:         "http://foo.bar.com",
			description: "some tarball",
		})
		Expect(found).To(BeFalse())
	})

	It("saves files under the base path named with their URL sha1 and tarball sha1", func() {
		fs.WriteFileString("source-path", "")

		err := cache.Save("source-path", &fakeSource{
			sha1:        "fake-sha1",
			url:         "http://foo.bar.com",
			description: "some tarball",
		})
		Expect(err).ToNot(HaveOccurred())
		// echo -n "http://foo.bar.com" | openssl sha1 -> 587cd74a86333e7f1ebca70474a1f4456e4b5d3e
		Expect(cache.Path(&fakeSource{
			sha1:        "fake-sha1",
			url:         "http://foo.bar.com",
			description: "some tarball",
		})).To(Equal(filepath.Join("/", "fake-base-path", "587cd74a86333e7f1ebca70474a1f4456e4b5d3e-fake-sha1")))
		Expect(fs.FileExists(filepath.Join("/", "fake-base-path", "587cd74a86333e7f1ebca70474a1f4456e4b5d3e-fake-sha1"))).To(BeTrue())
	})

	It("saves files across devices when necessary", func() {
		fs.RenameError = &os.LinkError{
			Err: syscall.Errno(0x12),
		}
		fs.WriteFileString("source-path", "")

		err := cache.Save("source-path", &fakeSource{
			sha1:        "fake-sha1",
			url:         "http://foo.bar.com",
			description: "some tarball",
		})
		Expect(err).ToNot(HaveOccurred())
	})
})
