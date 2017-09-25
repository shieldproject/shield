package fileutil

import (
	"os"
	"path/filepath"

	"github.com/cloudfoundry/gofileutils/glob"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

const cpCopierLogTag = "cpCopier"

type cpCopier struct {
	fs        boshsys.FileSystem
	cmdRunner boshsys.CmdRunner
	logger    boshlog.Logger
}

func NewCpCopier(
	cmdRunner boshsys.CmdRunner,
	fs boshsys.FileSystem,
	logger boshlog.Logger,
) Copier {
	return cpCopier{fs: fs, cmdRunner: cmdRunner, logger: logger}
}

func (c cpCopier) FilteredCopyToTemp(dir string, filters []string) (string, error) {
	filters = c.convertDirectoriesToGlobs(dir, filters)

	dirGlob := glob.NewDir(dir)
	filesToCopy, err := dirGlob.Glob(filters...)
	if err != nil {
		return "", bosherr.WrapError(err, "Finding files matching filters")
	}

	return c.tryInTempDir(func(tempDir string) error {
		for _, relativePath := range filesToCopy {
			src := filepath.Join(dir, relativePath)
			dst := filepath.Join(tempDir, relativePath)

			fileInfo, err := os.Stat(src)
			if err != nil {
				return bosherr.WrapErrorf(err, "Getting file info for '%s'", src)
			}

			if !fileInfo.IsDir() {
				err = c.cp(src, dst, tempDir)
			}

			if err != nil {
				return err
			}
		}

		err = c.fs.Chmod(tempDir, os.FileMode(0755))
		if err != nil {
			bosherr.WrapError(err, "Fixing permissions on temp dir")
		}

		return nil
	})
}

func (c cpCopier) tryInTempDir(fn func(string) error) (string, error) {
	tempDir, err := c.fs.TempDir("bosh-platform-commands-cpCopier-FilteredCopyToTemp")
	if err != nil {
		return "", bosherr.WrapError(err, "Creating temporary directory")
	}

	err = fn(tempDir)
	if err != nil {
		c.CleanUp(tempDir)
		return "", err
	}

	return tempDir, nil
}

func (c cpCopier) CleanUp(tempDir string) {
	err := c.fs.RemoveAll(tempDir)
	if err != nil {
		c.logger.Error(cpCopierLogTag, "Failed to clean up temporary directory %s: %#v", tempDir, err)
	}
}

func (c cpCopier) convertDirectoriesToGlobs(dir string, filters []string) []string {
	convertedFilters := []string{}
	for _, filter := range filters {
		src := filepath.Join(dir, filter)
		fileInfo, err := os.Stat(src)
		if err == nil && fileInfo.IsDir() {
			convertedFilters = append(convertedFilters, filepath.Join(filter, "**", "*"))
		} else {
			convertedFilters = append(convertedFilters, filter)
		}
	}

	return convertedFilters
}

func (c cpCopier) cp(src, dst, tempDir string) error {
	containingDir := filepath.Dir(dst)
	err := c.fs.MkdirAll(containingDir, os.ModePerm)
	if err != nil {
		return bosherr.WrapErrorf(err, "Making destination directory '%s' for '%s'", containingDir, src)
	}

	// Golang does not have a way of copying files and preserving file info...
	_, _, _, err = c.cmdRunner.RunCommand("cp", "-p", src, dst)
	if err != nil {
		c.CleanUp(tempDir)
		return bosherr.WrapError(err, "Shelling out to cp")
	}

	return nil
}
