package cmd

import (
	"github.com/cloudfoundry/bosh-cli/crypto"
	boshrel "github.com/cloudfoundry/bosh-cli/release"
	boshjob "github.com/cloudfoundry/bosh-cli/release/job"
	"github.com/cloudfoundry/bosh-cli/release/license"
	boshpkg "github.com/cloudfoundry/bosh-cli/release/pkg"
	boshui "github.com/cloudfoundry/bosh-cli/ui"
	crypto2 "github.com/cloudfoundry/bosh-utils/crypto"
	boshfu "github.com/cloudfoundry/bosh-utils/fileutil"
)

type Sha2ifyReleaseCmd struct {
	reader                boshrel.Reader
	writer                boshrel.Writer
	digestCalculator      crypto.DigestCalculator
	mv                    boshfu.Mover
	archiveFilePathReader crypto2.ArchiveDigestFilePathReader
	ui                    boshui.UI
}

func NewSha2ifyReleaseCmd(
	reader boshrel.Reader,
	writer boshrel.Writer,
	digestCalculator crypto.DigestCalculator,
	mv boshfu.Mover,
	archiveFilePathReader crypto2.ArchiveDigestFilePathReader,
	ui boshui.UI,
) Sha2ifyReleaseCmd {
	return Sha2ifyReleaseCmd{
		reader:           reader,
		writer:           writer,
		digestCalculator: digestCalculator,
		mv:               mv,
		archiveFilePathReader: archiveFilePathReader,
		ui: ui,
	}
}

func (cmd Sha2ifyReleaseCmd) Run(args Sha2ifyReleaseArgs) error {
	release, err := cmd.reader.Read(args.Path)
	if err != nil {
		return err
	}

	sha2jobs := []*boshjob.Job{}
	for _, job := range release.Jobs() {
		sha2Job, err := job.RehashWithCalculator(cmd.digestCalculator, cmd.archiveFilePathReader)
		if err != nil {
			return err
		}
		sha2jobs = append(sha2jobs, sha2Job)
	}

	sha2CompiledPackages := []*boshpkg.CompiledPackage{}
	for _, compPkg := range release.CompiledPackages() {
		sha2CompiledPackage, err := compPkg.RehashWithCalculator(cmd.digestCalculator, cmd.archiveFilePathReader)
		if err != nil {
			return err
		}
		sha2CompiledPackages = append(sha2CompiledPackages, sha2CompiledPackage)
	}

	sha2packages := []*boshpkg.Package{}
	for _, pkg := range release.Packages() {
		sha2Pkg, err := pkg.RehashWithCalculator(cmd.digestCalculator, cmd.archiveFilePathReader)
		if err != nil {
			return err
		}
		sha2packages = append(sha2packages, sha2Pkg)
	}

	var sha2License *license.License
	releaseLicense := release.License()
	if releaseLicense != nil {
		sha2License, err = releaseLicense.RehashWithCalculator(cmd.digestCalculator, cmd.archiveFilePathReader)
		if err != nil {
			return err
		}
	}

	sha2release := release.CopyWith(sha2jobs, sha2packages, sha2License, sha2CompiledPackages)

	tmpWriterPath, err := cmd.writer.Write(sha2release, nil)
	if err != nil {
		return err
	}

	err = cmd.mv.Move(tmpWriterPath, args.Destination.ExpandedPath)
	if err != nil {
		return err
	}

	ReleaseTables{Release: sha2release, ArchivePath: args.Destination.ExpandedPath}.Print(cmd.ui)

	return nil
}
