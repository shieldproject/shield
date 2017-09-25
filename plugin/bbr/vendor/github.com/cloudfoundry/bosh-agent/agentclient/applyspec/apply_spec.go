package applyspec

// ApplySpec is the transport layer model for communicating instance state to the bosh-agent.
// The format is suboptimal for its current usage. :(
type ApplySpec struct {
	Deployment       string `json:"deployment"`
	Name             string `json:"name"`
	Index            int    `json:"index"`
	NodeID           string `json:"id"`
	AvailabilityZone string `json:"az"`
	// Packages is a map of package names to compiled package blob references
	Packages map[string]Blob `json:"packages"`
	// Networks is a map of network names to network interfaces.
	// The value type would ideally be a struct with IP, Type & CloudProperties, but the agent supports arbitrary key/value pairs. :(
	Networks                 map[string]interface{}       `json:"networks"`
	Job                      Job                          `json:"job"`
	RenderedTemplatesArchive RenderedTemplatesArchiveSpec `json:"rendered_templates_archive"`
	ConfigurationHash        string                       `json:"configuration_hash"`
}

// Blob is a reference to a named and versioned object, with an archive uploaded to the blobstore.
type Blob struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	SHA1        string `json:"sha1"`
	BlobstoreID string `json:"blobstore_id"`
}

// Job is a description of an instance, and the 'jobs' running on it.
// Naming uses the historical Job/Templates pattern for reverse compatibility.
// If/When we added support for future format versions, this should be flattened into the ApplySpec, with 'Templates' renamed to 'Jobs'.
type Job struct {
	Name string `json:"name"`
	// Templates refer to release jobs, rendered specifically for this instance.
	// The SHA/BlobstoreID of the 'Templates' are currently being ignored by the bosh-agent,
	// because the RenderedTemplatesArchive contains the aggregate of all rendered jobs' templates.
	Templates []Blob `json:"templates"`
}

// RenderedTemplatesArchiveSpec is a reference to the aggregate job template archive, uploaded to the blobstore.
type RenderedTemplatesArchiveSpec struct {
	BlobstoreID string `json:"blobstore_id"`
	SHA1        string `json:"sha1"`
}
