package release_test

import (
	"errors"
	"os"

	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/release"
	fakerel "github.com/cloudfoundry/bosh-cli/release/releasefakes"
)

var _ = Describe("MultiReader", func() {
	Describe("Read", func() {
		var (
			archiveReader  *fakerel.FakeReader
			manifestReader *fakerel.FakeReader
			dirReader      *fakerel.FakeReader
			fs             *fakesys.FakeFileSystem
			reader         MultiReader

			readRelease *fakerel.FakeRelease
		)

		BeforeEach(func() {
			archiveReader = &fakerel.FakeReader{}
			manifestReader = &fakerel.FakeReader{}
			dirReader = &fakerel.FakeReader{}
			opts := MultiReaderOpts{
				ArchiveReader:  archiveReader,
				ManifestReader: manifestReader,
				DirReader:      dirReader,
			}
			fs = fakesys.NewFakeFileSystem()
			reader = NewMultiReader(opts, fs)

			readRelease = &fakerel.FakeRelease{
				NameStub: func() string { return "name" },
			}
		})

		It("uses manifest reader when path points to a yml file", func() {
			manifestReader.ReadReturns(readRelease, nil)

			release, err := reader.Read("/release.yml")
			Expect(err).ToNot(HaveOccurred())
			Expect(release).To(Equal(readRelease))

			manifestReader.ReadReturns(nil, errors.New("fake-err"))

			_, err = reader.Read("/release.yml")
			Expect(err).To(Equal(errors.New("fake-err")))
		})

		It("uses dir reader when path is a directory", func() {
			fs.MkdirAll("/release", os.ModePerm)

			dirReader.ReadReturns(readRelease, nil)

			release, err := reader.Read("/release")
			Expect(err).ToNot(HaveOccurred())
			Expect(release).To(Equal(readRelease))

			dirReader.ReadReturns(nil, errors.New("fake-err"))

			_, err = reader.Read("/release")
			Expect(err).To(Equal(errors.New("fake-err")))
		})

		It("uses archive reader when path is not a yml file or a directory", func() {
			fs.WriteFileString("/release-archive", "archive")

			archiveReader.ReadReturns(readRelease, nil)

			release, err := reader.Read("/release-archive")
			Expect(err).ToNot(HaveOccurred())
			Expect(release).To(Equal(readRelease))

			archiveReader.ReadReturns(nil, errors.New("fake-err"))

			_, err = reader.Read("/release-archive")
			Expect(err).To(Equal(errors.New("fake-err")))
		})

		It("returns an error when stat-ing path fails", func() {
			fs.RegisterOpenFile("/release", &fakesys.FakeFile{
				StatErr: errors.New("fake-err"),
			})

			_, err := reader.Read("/release")
			Expect(err).To(Equal(errors.New("fake-err")))
		})
	})
})
