package backup

type artifactIdentifier struct {
	name          string
	instanceName  string
	instanceIndex string
	hasCustomName bool
}

func (ai artifactIdentifier) Name() string {
	return ai.name
}

func (ai artifactIdentifier) InstanceName() string {
	return ai.instanceName
}

func (ai artifactIdentifier) InstanceIndex() string {
	return ai.instanceIndex
}

func (ai artifactIdentifier) HasCustomName() bool {
	return ai.hasCustomName
}

func makeCustomArtifactIdentifier(artifact artifactMetadata) artifactIdentifier {
	return artifactIdentifier{name: artifact.Name, hasCustomName: true}
}
func makeDefaultArtifactIdentifier(artifact artifactMetadata, inst *instanceMetadata) artifactIdentifier {
	return artifactIdentifier{name: artifact.Name, hasCustomName: false, instanceName: inst.Name, instanceIndex: inst.Index}
}
