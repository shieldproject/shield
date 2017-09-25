package job

import (
	"path"

	bireljobmanifest "github.com/cloudfoundry/bosh-init/release/job/manifest"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshcmd "github.com/cloudfoundry/bosh-utils/fileutil"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	"gopkg.in/yaml.v2"
)

type Reader interface {
	Read() (Job, error)
}

type reader struct {
	archivePath      string
	extractedJobPath string
	extractor        boshcmd.Compressor
	fs               boshsys.FileSystem
}

func NewReader(
	archivePath string,
	extractedJobPath string,
	extractor boshcmd.Compressor,
	fs boshsys.FileSystem,
) Reader {
	return &reader{
		archivePath:      archivePath,
		extractedJobPath: extractedJobPath,
		extractor:        extractor,
		fs:               fs,
	}
}

func (r *reader) Read() (Job, error) {
	err := r.extractor.DecompressFileToDir(r.archivePath, r.extractedJobPath, boshcmd.CompressorOptions{})
	if err != nil {
		return Job{}, bosherr.WrapErrorf(err,
			"Extracting job archive '%s'",
			r.archivePath)
	}

	jobManifestPath := path.Join(r.extractedJobPath, "job.MF")
	jobManifestBytes, err := r.fs.ReadFile(jobManifestPath)
	if err != nil {
		return Job{}, bosherr.WrapErrorf(err, "Reading job manifest '%s'", jobManifestPath)
	}

	var jobManifest bireljobmanifest.Manifest
	err = yaml.Unmarshal(jobManifestBytes, &jobManifest)
	if err != nil {
		return Job{}, bosherr.WrapError(err, "Parsing job manifest")
	}

	job := Job{
		Name:          jobManifest.Name,
		Templates:     jobManifest.Templates,
		PackageNames:  jobManifest.Packages,
		ExtractedPath: r.extractedJobPath,
	}

	jobProperties := make(map[string]PropertyDefinition, len(jobManifest.Properties))
	for propertyName, rawPropertyDef := range jobManifest.Properties {
		defaultValue, err := biproperty.Build(rawPropertyDef.Default)
		if err != nil {
			return Job{}, bosherr.WrapErrorf(err, "Parsing job '%s' property '%s' default: %#v", job.Name, propertyName, rawPropertyDef.Default)
		}
		jobProperties[propertyName] = PropertyDefinition{
			Description: rawPropertyDef.Description,
			Default:     defaultValue,
		}
	}
	job.Properties = jobProperties

	return job, nil
}
