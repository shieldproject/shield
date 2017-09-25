package fileutil_test

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-utils/fileutil"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

var _ = Describe("genericCpCopier", func() {
	var (
		fs       boshsys.FileSystem
		cpCopier Copier
	)

	BeforeEach(func() {
		logger := boshlog.NewLogger(boshlog.LevelNone)
		fs = boshsys.NewOsFileSystem(logger)
		cpCopier = NewGenericCpCopier(fs, logger)
	})

	Describe("FilteredCopyToTemp", func() {
		filesInDir := func(dir string) []string {
			copiedFiles := []string{}
			err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					copiedFiles = append(copiedFiles, path)
				}
				return nil
			})

			Expect(err).ToNot(HaveOccurred())

			sort.Strings(copiedFiles)

			return copiedFiles
		}

		It("copies all regular files from filtered copy to temp", func() {
			srcDir := fixtureSrcDir()
			filters := []string{
				filepath.Join("**", "*.stdout.log"),
				"*.stderr.log",
				filepath.Join("**", "more.stderr.log"),
				filepath.Join("..", "some.config"),
				filepath.Join("some_directory", "**", "*"),
			}

			dstDir, err := cpCopier.FilteredCopyToTemp(srcDir, filters)
			Expect(err).ToNot(HaveOccurred())

			defer os.RemoveAll(dstDir)

			copiedFiles := filesInDir(dstDir)

			Expect(err).ToNot(HaveOccurred())

			Expect(copiedFiles[0:5]).To(Equal([]string{
				filepath.Join(dstDir, "app.stderr.log"),
				filepath.Join(dstDir, "app.stdout.log"),
				filepath.Join(dstDir, "other_logs", "more_logs", "more.stdout.log"),
				filepath.Join(dstDir, "other_logs", "other_app.stdout.log"),
				filepath.Join(dstDir, "some_directory", "sub_dir", "other_sub_dir", ".keep"),
			}))

			content, err := fs.ReadFileString(filepath.Join(dstDir, "app.stdout.log"))
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("this is app stdout"))

			content, err = fs.ReadFileString(filepath.Join(dstDir, "app.stderr.log"))
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("this is app stderr"))

			content, err = fs.ReadFileString(filepath.Join(dstDir, "other_logs", "other_app.stdout.log"))
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("this is other app stdout"))

			content, err = fs.ReadFileString(filepath.Join(dstDir, "other_logs", "more_logs", "more.stdout.log"))
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("this is more stdout"))

			Expect(fs.FileExists(filepath.Join(dstDir, "some_directory"))).To(BeTrue())
			Expect(fs.FileExists(filepath.Join(dstDir, "some_directory", "sub_dir"))).To(BeTrue())
			Expect(fs.FileExists(filepath.Join(dstDir, "some_directory", "sub_dir", "other_sub_dir"))).To(BeTrue())

			_, err = fs.ReadFile(filepath.Join(dstDir, "other_logs", "other_app.stderr.log"))
			Expect(err).To(HaveOccurred())

			_, err = fs.ReadFile(filepath.Join(dstDir, "..", "some.config"))
			Expect(err).To(HaveOccurred())
		})

		It("copies all symlinked files from filtered copy to temp", func() {
			if runtime.GOOS == "windows" {
				Skip("Pending on Windows, relative symlinks are not supported")
			}

			srcDir := fixtureSrcDir()
			symlinkPath, err := createTestSymlink()
			Expect(err).To(Succeed())
			defer os.Remove(symlinkPath)

			filters := []string{
				filepath.Join("**", "*.stdout.log"),
				"*.stderr.log",
				filepath.Join("**", "more.stderr.log"),
				filepath.Join("..", "some.config"),
				filepath.Join("some_directory", "**", "*"),
			}

			dstDir, err := cpCopier.FilteredCopyToTemp(srcDir, filters)
			Expect(err).ToNot(HaveOccurred())

			defer os.RemoveAll(dstDir)

			copiedFiles := filesInDir(dstDir)

			Expect(err).ToNot(HaveOccurred())

			Expect(copiedFiles[5:]).To(Equal([]string{
				filepath.Join(dstDir, "symlink_dir", "app.stdout.log"),
				filepath.Join(dstDir, "symlink_dir", "sub_dir", "sub_app.stdout.log"),
			}))
		})

		Describe("changing permissions", func() {
			BeforeEach(func() {
				if runtime.GOOS == "windows" {
					// https://golang.org/src/os/path_test.go#L124
					Skip("Pending on Windows, chmod is not supported")
				}
			})

			It("fixes permissions on destination directory", func() {
				srcDir := fixtureSrcDir()
				filters := []string{
					"**/*",
				}

				dstDir, err := cpCopier.FilteredCopyToTemp(srcDir, filters)
				Expect(err).ToNot(HaveOccurred())

				defer os.RemoveAll(dstDir)

				tarDirStat, err := os.Stat(dstDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(os.FileMode(0755)).To(Equal(tarDirStat.Mode().Perm()))
			})
		})

		It("copies the content of directories when specified as a filter", func() {
			srcDir := fixtureSrcDir()
			filters := []string{
				"some_directory",
			}

			dstDir, err := cpCopier.FilteredCopyToTemp(srcDir, filters)
			Expect(err).ToNot(HaveOccurred())

			defer os.RemoveAll(dstDir)

			copiedFiles := filesInDir(dstDir)

			Expect(copiedFiles).To(Equal([]string{
				filepath.Join(dstDir, "some_directory", "sub_dir", "other_sub_dir", ".keep"),
			}))
		})
	})

	Describe("CleanUp", func() {
		It("cleans up", func() {
			tempDir := filepath.Join(os.TempDir(), "test-copier-cleanup")
			fs.MkdirAll(tempDir, os.ModePerm)

			cpCopier.CleanUp(tempDir)

			_, err := os.Stat(tempDir)
			Expect(err).To(HaveOccurred())
		})
	})
})
