package state

import (
	biagentclient "github.com/cloudfoundry/bosh-agent/agentclient"
	biblobstore "github.com/cloudfoundry/bosh-init/blobstore"
	birelpkg "github.com/cloudfoundry/bosh-init/release/pkg"
	bistatepkg "github.com/cloudfoundry/bosh-init/state/pkg"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type remotePackageCompiler struct {
	blobstore   biblobstore.Blobstore
	agentClient biagentclient.AgentClient
	packageRepo bistatepkg.CompiledPackageRepo
}

func NewRemotePackageCompiler(blobstore biblobstore.Blobstore, agentClient biagentclient.AgentClient, packageRepo bistatepkg.CompiledPackageRepo) bistatepkg.Compiler {
	return &remotePackageCompiler{
		blobstore:   blobstore,
		agentClient: agentClient,
		packageRepo: packageRepo,
	}
}

func (c *remotePackageCompiler) Compile(releasePackage *birelpkg.Package) (record bistatepkg.CompiledPackageRecord, isAlreadyCompiled bool, err error) {

	blobID, err := c.blobstore.Add(releasePackage.ArchivePath)
	if err != nil {
		return bistatepkg.CompiledPackageRecord{}, false, bosherr.WrapErrorf(err, "Adding release package archive '%s' to blobstore", releasePackage.ArchivePath)
	}

	packageSource := biagentclient.BlobRef{
		Name:        releasePackage.Name,
		Version:     releasePackage.Fingerprint,
		SHA1:        releasePackage.SHA1,
		BlobstoreID: blobID,
	}

	// If the package is a source package
	if releasePackage.Stemcell == "" {
		// Resolve dependencies from map of previously compiled packages.
		// Only install the package's immediate dependencies when compiling (not all transitive dependencies).
		packageDependencies := make([]biagentclient.BlobRef, len(releasePackage.Dependencies), len(releasePackage.Dependencies))
		for i, dependency := range releasePackage.Dependencies {
			compiledPackageRecord, found, err := c.packageRepo.Find(*dependency)
			if err != nil {
				return record, false, bosherr.WrapErrorf(
					err,
					"Finding compiled package '%s/%s' as dependency for '%s/%s'",
					dependency.Name,
					dependency.Fingerprint,
					releasePackage.Name,
					releasePackage.Fingerprint,
				)
			}
			if !found {
				return record, false, bosherr.Errorf(
					"Remote compilation failure: Package '%s/%s' requires package '%s/%s', but it has not been compiled",
					releasePackage.Name,
					releasePackage.Fingerprint,
					dependency.Name,
					dependency.Fingerprint,
				)
			}
			packageDependencies[i] = biagentclient.BlobRef{
				Name:        dependency.Name,
				Version:     dependency.Fingerprint,
				BlobstoreID: compiledPackageRecord.BlobID,
				SHA1:        compiledPackageRecord.BlobSHA1,
			}
		}

		compiledPackageRef, err := c.agentClient.CompilePackage(packageSource, packageDependencies)
		if err != nil {
			return record, false, bosherr.WrapErrorf(err, "Remotely compiling package '%s' with the agent", releasePackage.Name)
		}
		record = bistatepkg.CompiledPackageRecord{
			BlobID:   compiledPackageRef.BlobstoreID,
			BlobSHA1: compiledPackageRef.SHA1,
		}
	} else {
		// If it is a compiled package
		record = bistatepkg.CompiledPackageRecord{
			BlobID:   blobID,
			BlobSHA1: releasePackage.SHA1,
		}
		isAlreadyCompiled = true
	}

	err = c.packageRepo.Save(*releasePackage, record)
	if err != nil {
		return record, isAlreadyCompiled, bosherr.WrapErrorf(err, "Saving compiled package record %#v of package %#v", record, releasePackage)
	}

	return record, isAlreadyCompiled, nil
}
