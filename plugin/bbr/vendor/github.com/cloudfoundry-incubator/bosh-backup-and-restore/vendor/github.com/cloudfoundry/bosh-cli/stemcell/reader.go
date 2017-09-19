package stemcell

import (
	"github.com/pivotal-golang/yaml"
	"path/filepath"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshcmd "github.com/cloudfoundry/bosh-utils/fileutil"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type manifest struct {
	Name            string
	Version         string
	OS              string `yaml:"operating_system"`
	SHA1            string
	BoshProtocol    string                      `yaml:"bosh_protocol"`
	CloudProperties map[interface{}]interface{} `yaml:"cloud_properties"`
}

// Reader reads a stemcell tarball and returns a stemcell object containing
// parsed information (e.g. version, name)
type Reader interface {
	Read(stemcellTarballPath string, extractedPath string) (ExtractedStemcell, error)
}

type reader struct {
	compressor boshcmd.Compressor
	fs         boshsys.FileSystem
}

func NewReader(compressor boshcmd.Compressor, fs boshsys.FileSystem) Reader {
	return reader{compressor: compressor, fs: fs}
}

func (s reader) Read(stemcellTarballPath string, extractedPath string) (ExtractedStemcell, error) {
	err := s.compressor.DecompressFileToDir(stemcellTarballPath, extractedPath, boshcmd.CompressorOptions{})
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Extracting stemcell from '%s' to '%s'", stemcellTarballPath, extractedPath)
	}

	var rawManifest manifest
	manifestPath := filepath.Join(extractedPath, "stemcell.MF")

	manifestContents, err := s.fs.ReadFile(manifestPath)
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Reading stemcell manifest '%s'", manifestPath)
	}

	err = yaml.Unmarshal(manifestContents, &rawManifest)
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Parsing stemcell manifest: %s", manifestContents)
	}

	manifest := Manifest{
		Name:         rawManifest.Name,
		Version:      rawManifest.Version,
		OS:           rawManifest.OS,
		SHA1:         rawManifest.SHA1,
		BoshProtocol: rawManifest.BoshProtocol,
	}

	cloudProperties, err := biproperty.BuildMap(rawManifest.CloudProperties)
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Parsing stemcell cloud_properties: %#v", rawManifest.CloudProperties)
	}
	manifest.CloudProperties = cloudProperties

	stemcell := NewExtractedStemcell(
		manifest,
		extractedPath,
		s.compressor,
		s.fs,
	)

	return stemcell, nil
}
