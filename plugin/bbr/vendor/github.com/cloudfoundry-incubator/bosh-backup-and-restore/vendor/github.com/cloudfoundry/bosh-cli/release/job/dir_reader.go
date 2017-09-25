package job

import (
	"path/filepath"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"

	boshjobman "github.com/cloudfoundry/bosh-cli/release/job/manifest"
	. "github.com/cloudfoundry/bosh-cli/release/resource"
)

type DirReaderImpl struct {
	archiveFactory ArchiveFunc
	fs             boshsys.FileSystem
}

func NewDirReaderImpl(archiveFactory ArchiveFunc, fs boshsys.FileSystem) DirReaderImpl {
	return DirReaderImpl{archiveFactory: archiveFactory, fs: fs}
}

func (r DirReaderImpl) Read(path string) (*Job, error) {
	manifest, files, err := r.collectFiles(path)
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Collecting job files")
	}

	archive := r.archiveFactory(files, nil, nil)

	fp, err := archive.Fingerprint()
	if err != nil {
		return nil, err
	}

	job := NewJob(NewResource(manifest.Name, fp, archive))
	job.PackageNames = manifest.Packages
	// Does not read all manifest values...

	return job, nil
}

func (r DirReaderImpl) collectFiles(path string) (boshjobman.Manifest, []File, error) {
	var files []File

	specPath := filepath.Join(path, "spec")

	manifest, err := boshjobman.NewManifestFromPath(specPath, r.fs)
	if err != nil {
		return boshjobman.Manifest{}, nil, err
	}

	// Note that job's spec file is included (unlike for a package)
	// to capture differences in metadata of the job
	specFile := NewFile(specPath, path)
	specFile.RelativePath = "job.MF"
	files = append(files, specFile)

	monitPath := filepath.Join(path, "monit")

	if r.fs.FileExists(monitPath) {
		files = append(files, NewFile(monitPath, path))
	}

	for src, _ := range manifest.Templates {
		srcPath := filepath.Join(path, "templates", src)
		files = append(files, NewFile(srcPath, path))
	}

	return manifest, files, nil
}
