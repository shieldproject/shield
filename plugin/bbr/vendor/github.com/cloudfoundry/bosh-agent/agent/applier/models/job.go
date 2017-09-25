package models

type Job struct {
	Name    string
	Version string
	Source  Source

	// Packages that this job depends on; however,
	// currently it will contain packages from all jobs
	Packages []Package
}

func (s Job) BundleName() string {
	return s.Name
}

func (s Job) BundleVersion() string {
	if len(s.Version) == 0 {
		panic("Internal inconsistency: Expected job.Version to be non-empty")
	}

	// Job template is not unique per version because
	// Source contains files with interpolated values
	// which might be different across job versions.
	return s.Version + "-" + s.Source.Sha1.String()
}
