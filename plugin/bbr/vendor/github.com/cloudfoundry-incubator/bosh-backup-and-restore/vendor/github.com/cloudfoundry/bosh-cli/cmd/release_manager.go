package cmd

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	"github.com/cppforlife/go-patch/patch"
	semver "github.com/cppforlife/go-semi-semantic/version"

	boshdir "github.com/cloudfoundry/bosh-cli/director"
	boshtpl "github.com/cloudfoundry/bosh-cli/director/template"
	boshrel "github.com/cloudfoundry/bosh-cli/release"
)

type ReleaseManager struct {
	createReleaseCmd ReleaseCreatingCmd
	uploadReleaseCmd ReleaseUploadingCmd
}

type ReleaseUploadingCmd interface {
	Run(UploadReleaseOpts) error
}

type ReleaseCreatingCmd interface {
	Run(CreateReleaseOpts) (boshrel.Release, error)
}

func NewReleaseManager(
	createReleaseCmd ReleaseCreatingCmd,
	uploadReleaseCmd ReleaseUploadingCmd,
) ReleaseManager {
	return ReleaseManager{createReleaseCmd, uploadReleaseCmd}
}

func (m ReleaseManager) UploadReleases(bytes []byte) ([]byte, error) {
	manifest, err := boshdir.NewManifestFromBytes(bytes)
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Parsing manifest")
	}

	var opss patch.Ops

	for _, rel := range manifest.Releases {
		ops, err := m.createAndUploadRelease(rel)
		if err != nil {
			return nil, bosherr.WrapErrorf(err, "Processing release '%s/%s'", rel.Name, rel.Version)
		}

		opss = append(opss, ops)
	}

	tpl := boshtpl.NewTemplate(bytes)

	bytes, err = tpl.Evaluate(boshtpl.StaticVariables{}, opss, boshtpl.EvaluateOpts{})
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Updating manifest with created release versions")
	}

	return bytes, nil
}

func (m ReleaseManager) createAndUploadRelease(rel boshdir.ManifestRelease) (patch.Ops, error) {
	var ops patch.Ops

	if len(rel.URL) == 0 {
		return nil, nil
	}

	ver, err := semver.NewVersionFromString(rel.Version)
	if err != nil {
		return nil, err
	}

	uploadOpts := UploadReleaseOpts{
		Name:    rel.Name,
		Version: VersionArg(ver),

		Args: UploadReleaseArgs{URL: URLArg(rel.URL)},
		SHA1: rel.SHA1,
	}

	if rel.Version == "create" {
		createOpts := CreateReleaseOpts{
			Name:             rel.Name,
			Directory:        DirOrCWDArg{Path: uploadOpts.Args.URL.FilePath()},
			TimestampVersion: true,
			Force:            true,
		}

		release, err := m.createReleaseCmd.Run(createOpts)
		if err != nil {
			return nil, err
		}

		uploadOpts = UploadReleaseOpts{Release: release}

		replaceOp := patch.ReplaceOp{
			// equivalent to /releases/name=?/version
			Path: patch.NewPointer([]patch.Token{
				patch.RootToken{},
				patch.KeyToken{Key: "releases"},
				patch.MatchingIndexToken{Key: "name", Value: rel.Name},
				patch.KeyToken{Key: "version"},
			}),
			Value: release.Version(),
		}

		removeUrlOp := patch.RemoveOp{
			Path: patch.NewPointer([]patch.Token{
				patch.RootToken{},
				patch.KeyToken{Key: "releases"},
				patch.MatchingIndexToken{Key: "name", Value: rel.Name},
				patch.KeyToken{Key: "url"},
			}),
		}

		ops = append(ops, replaceOp, removeUrlOp)
	}

	return ops, m.uploadReleaseCmd.Run(uploadOpts)
}
