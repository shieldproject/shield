package resource

import (
	"os"
	"path/filepath"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshcmd "github.com/cloudfoundry/bosh-utils/fileutil"
	boshsys "github.com/cloudfoundry/bosh-utils/system"

	bicrypto "github.com/cloudfoundry/bosh-cli/crypto"
)

type ArchiveImpl struct {
	files            []File
	prepFiles        []File
	additionalChunks []string
	releaseDirPath   string

	fingerprinter    Fingerprinter
	compressor       boshcmd.Compressor
	digestCalculator bicrypto.DigestCalculator
	cmdRunner        boshsys.CmdRunner
	fs               boshsys.FileSystem
}

func NewArchiveImpl(
	files []File,
	prepFiles []File,
	additionalChunks []string,
	releaseDirPath string,
	fingerprinter Fingerprinter,
	compressor boshcmd.Compressor,
	digestCalculator bicrypto.DigestCalculator,
	cmdRunner boshsys.CmdRunner,
	fs boshsys.FileSystem,
) ArchiveImpl {
	return ArchiveImpl{
		files:            files,
		prepFiles:        prepFiles,
		additionalChunks: additionalChunks,
		releaseDirPath:   releaseDirPath,

		fingerprinter:    fingerprinter,
		compressor:       compressor,
		digestCalculator: digestCalculator,
		cmdRunner:        cmdRunner,
		fs:               fs,
	}
}

func (a ArchiveImpl) Fingerprint() (string, error) {
	fp, err := a.fingerprinter.Calculate(a.files, a.additionalChunks)
	if err != nil {
		return "", bosherr.WrapErrorf(err, "Fingerprinting source files")
	}

	return fp, nil
}

func (a ArchiveImpl) Build(expectedFp string) (string, string, error) {
	stagingDir, err := a.fs.TempDir("bosh-resource-archive")
	if err != nil {
		return "", "", bosherr.WrapError(err, "Creating staging directory")
	}

	defer func() {
		_ = a.fs.RemoveAll(stagingDir)
	}()

	for _, file := range a.files {
		err := a.copyFile(file, stagingDir)
		if err != nil {
			return "", "", bosherr.WrapError(err, "Copying into staging directory")
		}
	}

	stagingFp, err := a.buildStagingArchive(stagingDir).Fingerprint()
	if err != nil {
		return "", "", bosherr.WrapError(err, "Fingerprinting staged files")
	}

	if expectedFp != stagingFp {
		return "", "", bosherr.Errorf(
			"Expected source ('%s') and staging ('%s') fingerprints to match", expectedFp, stagingFp)
	}

	err = a.runPrepScripts(stagingDir)
	if err != nil {
		return "", "", bosherr.WrapError(err, "Running prep scripts")
	}

	archivePath, err := a.compressor.CompressFilesInDir(stagingDir)
	if err != nil {
		return "", "", bosherr.WrapError(err, "Compressing staging directory")
	}

	//generation of digest string
	archiveSHA1, err := a.digestCalculator.Calculate(archivePath)
	if err != nil {
		_ = a.compressor.CleanUp(archivePath)
		return "", "", bosherr.WrapError(err, "Calculating archive SHA1")
	}

	return archivePath, archiveSHA1, nil
}

func (a ArchiveImpl) runPrepScripts(stagingDir string) error {
	for _, prepFile := range a.prepFiles {
		// No need to copy into staging dir since we expect prep script to be in files

		cmd := boshsys.Command{
			Name: "bash",
			Args: []string{"-x", prepFile.Path},

			WorkingDir:     stagingDir,
			UseIsolatedEnv: false,

			Env: map[string]string{
				"BUILD_DIR":   stagingDir,
				"RELEASE_DIR": a.releaseDirPath,
			},
		}

		_, _, _, err := a.cmdRunner.RunComplexCommand(cmd)
		if err != nil {
			return err
		}

		// Arguably we should not remove the prep script
		err = a.fs.RemoveAll(filepath.Join(stagingDir, prepFile.RelativePath))
		if err != nil {
			return bosherr.WrapError(
				err, "Removing prep scrpt from staging directory")
		}
	}

	return nil
}

func (a ArchiveImpl) copyFile(sourceFile File, stagingDir string) error {
	dstPath := filepath.Join(stagingDir, sourceFile.RelativePath)
	dstDir := filepath.Dir(dstPath)

	err := a.fs.MkdirAll(dstDir, os.ModePerm)
	if err != nil {
		return err
	}

	sourceDirStat, err := a.fs.Lstat(filepath.Dir(sourceFile.Path))
	if err != nil {
		return err
	}

	err = a.fs.Chmod(dstDir, sourceDirStat.Mode())
	if err != nil {
		return err
	}

	sourceFileStat, err := a.fs.Lstat(sourceFile.Path)
	if err != nil {
		return err
	}

	if sourceFileStat.Mode()&os.ModeSymlink != 0 {
		symlinkTarget, err := a.fs.Readlink(sourceFile.Path)
		if err != nil {
			return err
		}

		return a.fs.Symlink(symlinkTarget, dstPath)
	} else {
		err = a.fs.CopyFile(sourceFile.Path, dstPath)
		if err != nil {
			return err
		}

		// Be very explicit about changing permissions for copied file
		// Only pay attention to whether the source file is executable
		return a.fs.Chmod(dstPath, getFilePerms(sourceFileStat))
	}
}

func getFilePerms(stat os.FileInfo) os.FileMode {
	if (stat.Mode() | 0100) == stat.Mode() {
		return 0755
	} else {
		return 0644
	}
}

func (a ArchiveImpl) buildStagingArchive(stagingDir string) Archive {
	var stagingFiles []File

	for _, file := range a.files {
		stagingFiles = append(stagingFiles, file.WithNewDir(stagingDir))
	}

	// Initialize with bare minimum deps so that fingerprinting can be performed
	return NewArchiveImpl(stagingFiles, nil, a.additionalChunks, "", a.fingerprinter, nil, nil, nil, nil)
}
