package pkg_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/release/pkg"
	. "github.com/cloudfoundry/bosh-cli/release/resource"

	fakeres "github.com/cloudfoundry/bosh-cli/release/resource/resourcefakes"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("DirReaderImpl", func() {
	var (
		collectedFiles     []File
		collectedPrepFiles []File
		collectedChunks    []string
		archive            *fakeres.FakeArchive
		fs                 *fakesys.FakeFileSystem
		reader             DirReaderImpl
	)

	BeforeEach(func() {
		archive = &fakeres.FakeArchive{}
		archiveFactory := func(files, prepFiles []File, chunks []string) Archive {
			collectedFiles = files
			collectedPrepFiles = prepFiles
			collectedChunks = chunks
			return archive
		}
		fs = fakesys.NewFakeFileSystem()

		reader = NewDirReaderImpl(archiveFactory, filepath.Join("/", "src"), filepath.Join("/", "blobs"), fs)
	})

	Describe("Read", func() {
		It("returns a package with the details collected from a directory", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "spec"), `---
name: name
dependencies: [pkg1, pkg2]
files: [in-file1, in-file2]
excluded_files: [ex-file1, ex-file2]
`)

			fs.WriteFileString(filepath.Join("/", "dir", "packaging"), "")
			fs.WriteFileString(filepath.Join("/", "src", "in-file1"), "")
			fs.WriteFileString(filepath.Join("/", "src", "in-file2"), "")
			fs.SetGlob(filepath.Join("/", "src", "in-file1"), []string{filepath.Join("/", "src", "in-file1")})
			fs.SetGlob(filepath.Join("/", "src", "in-file2"), []string{filepath.Join("/", "src", "in-file2")})

			archive.FingerprintReturns("fp", nil)

			pkg, err := reader.Read(filepath.Join("/", "dir"))
			Expect(err).NotTo(HaveOccurred())
			Expect(pkg).To(Equal(NewPackage(NewResource("name", "fp", archive), []string{"pkg1", "pkg2"})))

			Expect(collectedFiles).To(ConsistOf(
				// does not include spec
				File{Path: filepath.Join("/", "dir", "packaging"), DirPath: filepath.Join("/", "dir"), RelativePath: "packaging", ExcludeMode: true},
				File{Path: filepath.Join("/", "src", "in-file1"), DirPath: filepath.Join("/", "src"), RelativePath: "in-file1"},
				File{Path: filepath.Join("/", "src", "in-file2"), DirPath: filepath.Join("/", "src"), RelativePath: "in-file2"},
			))

			Expect(collectedPrepFiles).To(BeEmpty())
			Expect(collectedChunks).To(Equal([]string{"pkg1", "pkg2"}))
		})

		It("returns a package with the details with pre_packaging file", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "spec"), "name: name")
			fs.WriteFileString(filepath.Join("/", "dir", "packaging"), "")
			fs.WriteFileString(filepath.Join("/", "dir", "pre_packaging"), "")

			archive.FingerprintReturns("fp", nil)

			pkg, err := reader.Read(filepath.Join("/", "dir"))
			Expect(err).NotTo(HaveOccurred())
			Expect(pkg).To(Equal(NewPackage(NewResource("name", "fp", archive), nil)))

			Expect(collectedFiles).To(Equal([]File{
				File{Path: filepath.Join("/", "dir", "packaging"), DirPath: filepath.Join("/", "dir"), RelativePath: "packaging", ExcludeMode: true},
				File{Path: filepath.Join("/", "dir", "pre_packaging"), DirPath: filepath.Join("/", "dir"), RelativePath: "pre_packaging", ExcludeMode: true},
			}))

			Expect(collectedPrepFiles).To(Equal([]File{
				File{Path: filepath.Join("/", "dir", "pre_packaging"), DirPath: filepath.Join("/", "dir"), RelativePath: "pre_packaging", ExcludeMode: true},
			}))

			Expect(collectedChunks).To(BeEmpty())
		})

		It("returns a package with src files and blob files", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "spec"), `---
name: name
files: [in-file1, in-file2]
`)

			fs.WriteFileString(filepath.Join("/", "dir", "packaging"), "")
			fs.WriteFileString(filepath.Join("/", "dir", "pre_packaging"), "")
			fs.WriteFileString(filepath.Join("/", "src", "in-file1"), "")
			fs.WriteFileString(filepath.Join("/", "blobs", "in-file2"), "")

			fs.SetGlob(filepath.Join("/", "src", "in-file1"), []string{filepath.Join("/", "src", "in-file1")})
			fs.SetGlob(filepath.Join("/", "blobs", "in-file2"), []string{filepath.Join("/", "blobs", "in-file2")})

			archive.FingerprintReturns("fp", nil)

			pkg, err := reader.Read(filepath.Join("/", "dir"))
			Expect(err).NotTo(HaveOccurred())
			Expect(pkg).To(Equal(NewPackage(NewResource("name", "fp", archive), nil)))

			Expect(collectedFiles).To(ConsistOf([]File{
				File{Path: filepath.Join("/", "dir", "packaging"), DirPath: filepath.Join("/", "dir"), RelativePath: "packaging", ExcludeMode: true},
				File{Path: filepath.Join("/", "dir", "pre_packaging"), DirPath: filepath.Join("/", "dir"), RelativePath: "pre_packaging", ExcludeMode: true},
				File{Path: filepath.Join("/", "src", "in-file1"), DirPath: filepath.Join("/", "src"), RelativePath: "in-file1"},
				File{Path: filepath.Join("/", "blobs", "in-file2"), DirPath: filepath.Join("/", "blobs"), RelativePath: "in-file2"},
			}))

			Expect(collectedPrepFiles).To(Equal([]File{
				File{Path: filepath.Join("/", "dir", "pre_packaging"), DirPath: filepath.Join("/", "dir"), RelativePath: "pre_packaging", ExcludeMode: true},
			}))

			Expect(collectedChunks).To(BeEmpty())
		})

		It("prefers src files over blob files", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "spec"), `---
name: name
files: [in-file1, in-file2]
`)

			fs.WriteFileString(filepath.Join("/", "dir", "packaging"), "")
			fs.WriteFileString(filepath.Join("/", "src", "in-file1"), "")
			fs.WriteFileString(filepath.Join("/", "src", "in-file2"), "")
			fs.WriteFileString(filepath.Join("/", "blobs", "in-file2"), "")

			fs.SetGlob(filepath.Join("/", "src", "in-file1"), []string{filepath.Join("/", "src", "in-file1")})
			fs.SetGlob(filepath.Join("/", "src", "in-file2"), []string{filepath.Join("/", "src", "in-file2")})
			fs.SetGlob(filepath.Join("/", "blobs", "in-file2"), []string{filepath.Join("/", "blobs", "in-file2")})

			archive.FingerprintReturns("fp", nil)

			pkg, err := reader.Read(filepath.Join("/", "dir"))
			Expect(err).NotTo(HaveOccurred())
			Expect(pkg).To(Equal(NewPackage(NewResource("name", "fp", archive), nil)))

			Expect(collectedFiles).To(ConsistOf([]File{
				File{Path: filepath.Join("/", "dir", "packaging"), DirPath: filepath.Join("/", "dir"), RelativePath: "packaging", ExcludeMode: true},
				File{Path: filepath.Join("/", "src", "in-file1"), DirPath: filepath.Join("/", "src"), RelativePath: "in-file1"},
				File{Path: filepath.Join("/", "src", "in-file2"), DirPath: filepath.Join("/", "src"), RelativePath: "in-file2"},
			}))

			Expect(collectedPrepFiles).To(BeEmpty())
			Expect(collectedChunks).To(BeEmpty())
		})

		It("returns an error if glob doesnt match src or blob files", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "spec"), `---
name: name
files: [in-file1, in-file2, missing-file2]
`)

			fs.WriteFileString(filepath.Join("/", "dir", "packaging"), "")
			fs.WriteFileString(filepath.Join("/", "src", "in-file1"), "")
			fs.WriteFileString(filepath.Join("/", "blobs", "in-file2"), "")
			fs.SetGlob(filepath.Join("/", "src", "in-file1"), []string{filepath.Join("/", "src", "in-file1")})
			fs.SetGlob(filepath.Join("/", "blobs", "in-file2"), []string{filepath.Join("/", "blobs", "in-file2")})

			// Directories are not packageable
			fs.MkdirAll(filepath.Join("/", "src", "missing-file2"), os.ModePerm)
			fs.MkdirAll(filepath.Join("/", "blobs", "missing-file2"), os.ModePerm)
			fs.SetGlob(filepath.Join("/", "src", "missing-file2"), []string{filepath.Join("/", "src", "missing-file2")})
			fs.SetGlob(filepath.Join("/", "blobs", "missing-file2"), []string{filepath.Join("/", "blobs", "missing-file2")})

			archive.FingerprintReturns("fp", nil)

			_, err := reader.Read(filepath.Join("/", "dir"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Missing files for pattern 'missing-file2'"))
		})

		It("excludes files and blobs", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "spec"), `---
name: name
files: [in-file1, in-file2]
excluded_files: [ex-file1, ex-file2]
`)

			fs.WriteFileString(filepath.Join("/", "dir", "packaging"), "")
			fs.WriteFileString(filepath.Join("/", "src", "in-file1"), "")
			fs.WriteFileString(filepath.Join("/", "blobs", "in-file2"), "")

			fs.SetGlob(filepath.Join("/", "src", "in-file1"), []string{filepath.Join("/", "src", "in-file1")})
			fs.SetGlob(filepath.Join("/", "blobs", "in-file2"), []string{filepath.Join("/", "blobs", "in-file2")})
			fs.SetGlob(filepath.Join("/", "src", "ex-file1"), []string{filepath.Join("/", "src", "in-file1")})
			fs.SetGlob(filepath.Join("/", "blobs", "ex-file2"), []string{filepath.Join("/", "blobs", "in-file2")})

			archive.FingerprintReturns("fp", nil)

			pkg, err := reader.Read(filepath.Join("/", "dir"))
			Expect(err).NotTo(HaveOccurred())
			Expect(pkg).To(Equal(NewPackage(NewResource("name", "fp", archive), nil)))

			Expect(collectedFiles).To(Equal([]File{
				File{Path: filepath.Join("/", "dir", "packaging"), DirPath: filepath.Join("/", "dir"), RelativePath: "packaging", ExcludeMode: true},
			}))

			Expect(collectedPrepFiles).To(BeEmpty())
			Expect(collectedChunks).To(BeEmpty())
		})

		It("allows to only have packaging file and to exclude all files", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "spec"), `---
name: name
excluded_files: [ex-file1, ex-file2]
`)

			fs.WriteFileString(filepath.Join("/", "dir", "packaging"), "")

			archive.FingerprintReturns("fp", nil)

			pkg, err := reader.Read(filepath.Join("/", "dir"))
			Expect(err).NotTo(HaveOccurred())
			Expect(pkg).To(Equal(NewPackage(NewResource("name", "fp", archive), nil)))

			Expect(collectedFiles).To(Equal([]File{
				File{Path: filepath.Join("/", "dir", "packaging"), DirPath: filepath.Join("/", "dir"), RelativePath: "packaging", ExcludeMode: true},
			}))

			Expect(collectedPrepFiles).To(BeEmpty())
			Expect(collectedChunks).To(BeEmpty())
		})

		It("matches files in blobs directory even if glob also matches empty folders in src directory", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "spec"), `---
name: name
dependencies: [pkg1, pkg2]
files:
- "**/*"
excluded_files: [ex-file1, ex-file2]
`)
			fs.SetGlob(filepath.Join("/", "src", "**", "*"), []string{filepath.Join("/", "src", "directory")})

			err := fs.MkdirAll(filepath.Join("/", "src", "directory"), 0777)
			Expect(err).NotTo(HaveOccurred())

			err = fs.WriteFileString(filepath.Join("/", "dir", "packaging"), "")
			Expect(err).NotTo(HaveOccurred())

			err = fs.MkdirAll(filepath.Join("/", "src", "directory", "f1", "/"), 0777)
			Expect(err).NotTo(HaveOccurred())

			err = fs.WriteFileString(filepath.Join("/", "blobs", "directory", "f1"), "")
			Expect(err).NotTo(HaveOccurred())

			fs.SetGlob(filepath.Join("/", "blobs", "**", "*"), []string{filepath.Join("/", "blobs", "directory"), filepath.Join("/", "blobs", "directory", "f1")})
			fs.SetGlob(filepath.Join("/", "src", "**", "*"), []string{filepath.Join("/", "src", "directory"), filepath.Join("/", "src", "directory", "f1")})

			_, err = reader.Read(filepath.Join("/", "dir"))
			Expect(err).NotTo(HaveOccurred())

			Expect(collectedFiles).To(Equal([]File{
				File{Path: filepath.Join("/", "dir", "packaging"), DirPath: filepath.Join("/", "dir"), RelativePath: "packaging", ExcludeMode: true},
				File{Path: filepath.Join("/", "blobs", "directory", "f1"), DirPath: filepath.Join("/", "blobs"), RelativePath: filepath.Join("directory", "f1")},
			}))
		})

		It("returns error if packaging is included in specified files", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "spec"), "name: name\nfiles: [packaging]")

			fs.WriteFileString(filepath.Join("/", "dir", "packaging"), "")
			fs.WriteFileString(filepath.Join("/", "src", "packaging"), "")
			fs.SetGlob(filepath.Join("/", "src", "packaging"), []string{filepath.Join("/", "src", "packaging")})

			_, err := reader.Read(filepath.Join("/", "dir"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Expected special 'packaging' file to not be included via 'files' key for package 'name'"))
		})

		It("returns error if pre_packaging is included in specified files", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "spec"), "name: name\nfiles: [pre_packaging]")

			fs.WriteFileString(filepath.Join("/", "dir", "packaging"), "")
			fs.WriteFileString(filepath.Join("/", "dir", "pre_packaging"), "")
			fs.WriteFileString(filepath.Join("/", "src", "pre_packaging"), "")
			fs.SetGlob(filepath.Join("/", "src", "pre_packaging"), []string{filepath.Join("/", "src", "pre_packaging")})

			_, err := reader.Read(filepath.Join("/", "dir"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Expected special 'pre_packaging' file to not be included via 'files' key for package 'name'"))
		})

		It("returns error if spec file is not valid", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "spec"), `-`)

			_, err := reader.Read(filepath.Join("/", "dir"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Collecting package files"))
		})

		It("returns error if packaging file is not found", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "spec"), "name: name")

			_, err := reader.Read(filepath.Join("/", "dir"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Expected to find '" + filepath.Join("/", "dir", "packaging") + "' for package 'name'"))
		})

		globErrChecks := map[string]string{
			"src files dir (files)":            filepath.Join("/", "src", "file1"),
			"blobs files dir (files)":          filepath.Join("/", "blobs", "file1"),
			"src files dir (excluded files)":   filepath.Join("/", "src", "ex-file1"),
			"blobs files dir (excluded files)": filepath.Join("/", "blobs", "ex-file1"),
		}

		for desc, pattern := range globErrChecks {
			desc, pattern := desc, pattern // copy

			It(fmt.Sprintf("returns error if globbing '%s' fails", desc), func() {
				fs.WriteFileString(filepath.Join("/", "dir", "spec"), "name: name\nfiles: [file1]\nexcluded_files: [ex-file1]")
				fs.WriteFileString(filepath.Join("/", "dir", "packaging"), "")

				fs.WriteFileString(filepath.Join("/", "src", "file1"), "")
				fs.WriteFileString(filepath.Join("/", "blobs", "file1"), "")
				fs.SetGlob(filepath.Join("/", "src", "file1"), []string{filepath.Join("/", "src", "file1")})
				fs.SetGlob(filepath.Join("/", "blobs", "file1"), []string{filepath.Join("/", "blobs", "file1")})

				fs.GlobErrs[pattern] = errors.New("fake-err")

				_, err := reader.Read(filepath.Join("/", "dir"))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})
		}

		It("returns error if fingerprinting fails", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "spec"), "")
			fs.WriteFileString(filepath.Join("/", "dir", "packaging"), "")

			archive.FingerprintReturns("", errors.New("fake-err"))

			_, err := reader.Read(filepath.Join("/", "dir"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("include bad src symlinks", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "spec"), `---
name: name
files: [in-file1, in-file2]
`)

			fs.WriteFileString(filepath.Join("/", "dir", "packaging"), "")
			fs.WriteFileString(filepath.Join("/", "dir", "pre_packaging"), "")
			fs.Symlink(filepath.Join("/", "invalid", "path"), filepath.Join("/", "src", "in-file1"))
			fs.WriteFileString(filepath.Join("/", "blobs", "in-file2"), "")

			fs.SetGlob(filepath.Join("/", "src", "in-file1"), []string{filepath.Join("/", "src", "in-file1")})
			fs.SetGlob(filepath.Join("/", "blobs", "in-file2"), []string{filepath.Join("/", "blobs", "in-file2")})

			archive.FingerprintReturns("fp", nil)

			pkg, err := reader.Read(filepath.Join("/", "dir"))
			Expect(err).NotTo(HaveOccurred())
			Expect(pkg).To(Equal(NewPackage(NewResource("name", "fp", archive), nil)))

			Expect(collectedFiles).To(ConsistOf([]File{
				File{Path: filepath.Join("/", "dir", "packaging"), DirPath: filepath.Join("/", "dir"), RelativePath: "packaging", ExcludeMode: true},
				File{Path: filepath.Join("/", "dir", "pre_packaging"), DirPath: filepath.Join("/", "dir"), RelativePath: "pre_packaging", ExcludeMode: true},
				File{Path: filepath.Join("/", "src", "in-file1"), DirPath: filepath.Join("/", "src"), RelativePath: "in-file1"},
				File{Path: filepath.Join("/", "blobs", "in-file2"), DirPath: filepath.Join("/", "blobs"), RelativePath: "in-file2"},
			}))

			Expect(collectedPrepFiles).To(Equal([]File{
				File{Path: filepath.Join("/", "dir", "pre_packaging"), DirPath: filepath.Join("/", "dir"), RelativePath: "pre_packaging", ExcludeMode: true},
			}))

			Expect(collectedChunks).To(BeEmpty())
		})
	})
})
