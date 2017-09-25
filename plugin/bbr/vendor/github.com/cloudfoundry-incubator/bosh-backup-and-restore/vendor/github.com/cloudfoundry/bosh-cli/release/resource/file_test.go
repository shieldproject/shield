package resource_test

import (
	"path/filepath"
	"sort"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/release/resource"
)

var _ = Describe("NewFile", func() {
	It("returns file with relative path that does not start with separator", func() {
		file := NewFile(filepath.Join("/", "tmp", "file"), filepath.Join("/", "tmp"))
		Expect(file.Path).To(Equal(filepath.Join("/", "tmp", "file")))
		Expect(file.DirPath).To(Equal(filepath.Join("/", "tmp")))
		Expect(file.RelativePath).To(Equal("file"))
	})

	It("returns file with relative path when dir path ends with separator", func() {
		file := NewFile(filepath.Join("/", "tmp", "file"), filepath.Join("/", "tmp", "/"))
		Expect(file.Path).To(Equal(filepath.Join("/", "tmp", "file")))
		Expect(file.DirPath).To(Equal(filepath.Join("/", "tmp")))
		Expect(file.RelativePath).To(Equal("file"))
	})
})

var _ = Describe("File", func() {
	Describe("WithNewDir", func() {
		It("returns file as if it was from a different dir", func() {
			file := NewFile(filepath.Join("/", "tmp", "file"), filepath.Join("/", "tmp", "/")).WithNewDir(filepath.Join("/", "other"))
			Expect(file.Path).To(Equal(filepath.Join("/", "other", "file")))
			Expect(file.DirPath).To(Equal(filepath.Join("/", "other")))
			Expect(file.RelativePath).To(Equal("file"))
		})
	})
})

var _ = Describe("FileRelativePathSorting", func() {
	It("sorts files based on relative path", func() {
		file2 := NewFile(filepath.Join("/", "tmp", "file2"), filepath.Join("/", "tmp", "/"))
		file1 := NewFile(filepath.Join("/", "tmp", "file1"), filepath.Join("/", "tmp", "/"))
		file := NewFile(filepath.Join("/", "tmp", "file"), filepath.Join("/", "tmp", "/"))
		files := []File{file2, file1, file}
		sort.Sort(FileRelativePathSorting(files))
		Expect(files).To(Equal([]File{file, file1, file2}))
	})
})
