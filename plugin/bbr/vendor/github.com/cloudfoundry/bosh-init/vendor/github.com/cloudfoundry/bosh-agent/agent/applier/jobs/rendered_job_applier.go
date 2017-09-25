package jobs

import (
	"fmt"
	"os"
	"path"
	"strings"

	boshbc "github.com/cloudfoundry/bosh-agent/agent/applier/bundlecollection"
	models "github.com/cloudfoundry/bosh-agent/agent/applier/models"
	"github.com/cloudfoundry/bosh-agent/agent/applier/packages"
	boshjobsuper "github.com/cloudfoundry/bosh-agent/jobsupervisor"
	boshblob "github.com/cloudfoundry/bosh-utils/blobstore"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshcmd "github.com/cloudfoundry/bosh-utils/fileutil"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

const logTag = "renderedJobApplier"

type renderedJobApplier struct {
	jobsBc                 boshbc.BundleCollection
	jobSupervisor          boshjobsuper.JobSupervisor
	packageApplierProvider packages.ApplierProvider
	blobstore              boshblob.Blobstore
	compressor             boshcmd.Compressor
	fs                     boshsys.FileSystem
	logger                 boshlog.Logger
}

func NewRenderedJobApplier(
	jobsBc boshbc.BundleCollection,
	jobSupervisor boshjobsuper.JobSupervisor,
	packageApplierProvider packages.ApplierProvider,
	blobstore boshblob.Blobstore,
	compressor boshcmd.Compressor,
	fs boshsys.FileSystem,
	logger boshlog.Logger,
) Applier {
	return &renderedJobApplier{
		jobsBc:                 jobsBc,
		jobSupervisor:          jobSupervisor,
		packageApplierProvider: packageApplierProvider,
		blobstore:              blobstore,
		compressor:             compressor,
		fs:                     fs,
		logger:                 logger,
	}
}

func (s renderedJobApplier) Prepare(job models.Job) error {
	s.logger.Debug(logTag, "Preparing job %v", job)

	jobBundle, err := s.jobsBc.Get(job)
	if err != nil {
		return bosherr.WrapError(err, "Getting job bundle")
	}

	jobInstalled, err := jobBundle.IsInstalled()
	if err != nil {
		return bosherr.WrapError(err, "Checking if job is installed")
	}

	if !jobInstalled {
		err := s.downloadAndInstall(job, jobBundle)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *renderedJobApplier) Apply(job models.Job) error {
	s.logger.Debug(logTag, "Applying job %v", job)

	err := s.Prepare(job)
	if err != nil {
		return bosherr.WrapError(err, "Preparing job")
	}

	jobBundle, err := s.jobsBc.Get(job)
	if err != nil {
		return bosherr.WrapError(err, "Getting job bundle")
	}

	_, _, err = jobBundle.Enable()
	if err != nil {
		return bosherr.WrapError(err, "Enabling job")
	}

	return s.applyPackages(job)
}

func (s *renderedJobApplier) downloadAndInstall(job models.Job, jobBundle boshbc.Bundle) error {
	tmpDir, err := s.fs.TempDir("bosh-agent-applier-jobs-RenderedJobApplier-Apply")
	if err != nil {
		return bosherr.WrapError(err, "Getting temp dir")
	}

	defer func() {
		if err = s.fs.RemoveAll(tmpDir); err != nil {
			s.logger.Warn(logTag, "Failed to clean up temp directory: %s", err.Error())
		}
	}()

	file, err := s.blobstore.Get(job.Source.BlobstoreID, job.Source.Sha1)
	if err != nil {
		return bosherr.WrapError(err, "Getting job source from blobstore")
	}

	defer func() {
		if err = s.blobstore.CleanUp(file); err != nil {
			s.logger.Warn(logTag, "Failed to clean up blobstore blob: %s", err.Error())
		}
	}()

	err = s.compressor.DecompressFileToDir(file, tmpDir, boshcmd.CompressorOptions{})
	if err != nil {
		return bosherr.WrapError(err, "Decompressing files to temp dir")
	}

	binPath := path.Join(tmpDir, job.Source.PathInArchive, "bin") + "/"
	err = s.fs.Walk(path.Join(tmpDir, job.Source.PathInArchive), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		} else if info.IsDir() || strings.HasPrefix(path, binPath) {
			return s.fs.Chmod(path, os.FileMode(0755))
		} else {
			return s.fs.Chmod(path, os.FileMode(0644))
		}
	})
	if err != nil {
		return bosherr.WrapError(err, "Correcting file permissions")
	}

	_, _, err = jobBundle.Install(path.Join(tmpDir, job.Source.PathInArchive))
	if err != nil {
		return bosherr.WrapError(err, "Installing job bundle")
	}

	return nil
}

// applyPackages keeps job specific packages directory up-to-date with installed packages.
// (e.g. /var/vcap/jobs/job-a/packages/pkg-a has symlinks to /var/vcap/packages/pkg-a)
func (s *renderedJobApplier) applyPackages(job models.Job) error {
	packageApplier := s.packageApplierProvider.JobSpecific(job.Name)

	for _, pkg := range job.Packages {
		err := packageApplier.Apply(pkg)
		if err != nil {
			return bosherr.WrapErrorf(err, "Applying package %s for job %s", pkg.Name, job.Name)
		}
	}

	err := packageApplier.KeepOnly(job.Packages)
	if err != nil {
		return bosherr.WrapErrorf(err, "Keeping only needed packages for job %s", job.Name)
	}

	return nil
}

func (s *renderedJobApplier) Configure(job models.Job, jobIndex int) (err error) {
	s.logger.Debug(logTag, "Configuring job %v with index %d", job, jobIndex)

	jobBundle, err := s.jobsBc.Get(job)
	if err != nil {
		err = bosherr.WrapError(err, "Getting job bundle")
		return
	}

	fs, jobDir, err := jobBundle.GetInstallPath()
	if err != nil {
		err = bosherr.WrapError(err, "Looking up job directory")
		return
	}

	monitFilePath := path.Join(jobDir, "monit")
	if fs.FileExists(monitFilePath) {
		err = s.jobSupervisor.AddJob(job.Name, jobIndex, monitFilePath)
		if err != nil {
			err = bosherr.WrapError(err, "Adding monit configuration")
			return
		}
	}

	monitFilePaths, err := fs.Glob(path.Join(jobDir, "*.monit"))
	if err != nil {
		err = bosherr.WrapError(err, "Looking for additional monit files")
		return
	}

	for _, monitFilePath := range monitFilePaths {
		label := strings.Replace(path.Base(monitFilePath), ".monit", "", 1)
		subJobName := fmt.Sprintf("%s_%s", job.Name, label)

		err = s.jobSupervisor.AddJob(subJobName, jobIndex, monitFilePath)
		if err != nil {
			err = bosherr.WrapErrorf(err, "Adding additional monit configuration %s", label)
			return
		}
	}

	return nil
}

func (s *renderedJobApplier) KeepOnly(jobs []models.Job) error {
	s.logger.Debug(logTag, "Keeping only jobs %v", jobs)

	installedBundles, err := s.jobsBc.List()
	if err != nil {
		return bosherr.WrapError(err, "Retrieving installed bundles")
	}

	for _, installedBundle := range installedBundles {
		var shouldKeep bool

		for _, job := range jobs {
			jobBundle, err := s.jobsBc.Get(job)
			if err != nil {
				return bosherr.WrapError(err, "Getting job bundle")
			}

			if jobBundle == installedBundle {
				shouldKeep = true
				break
			}
		}

		if !shouldKeep {
			err = installedBundle.Disable()
			if err != nil {
				return bosherr.WrapError(err, "Disabling job bundle")
			}

			// If we uninstall the bundle first, and the disable failed (leaving the symlink),
			// then the next time bundle collection will not include bundle in its list
			// which means that symlink will never be deleted.
			err = installedBundle.Uninstall()
			if err != nil {
				return bosherr.WrapError(err, "Uninstalling job bundle")
			}
		}
	}

	return nil
}
