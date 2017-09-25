package manifest

type Manifest struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`

	CommitHash         string `yaml:"commit_hash"`
	UncommittedChanges bool   `yaml:"uncommitted_changes"`

	Jobs             []JobRef     `yaml:"jobs"`
	Packages         []PackageRef `yaml:"packages"`
	CompiledPackages []PackageRef `yaml:"compiled_packages"`
}

type JobRef struct {
	Name        string `yaml:"name"`
	Fingerprint string `yaml:"fingerprint"`
	SHA1        string `yaml:"sha1"`
}

type PackageRef struct {
	Name         string   `yaml:"name"`
	Fingerprint  string   `yaml:"fingerprint"`
	SHA1         string   `yaml:"sha1"`
	Stemcell     string   `yaml:"stemcell"`
	Dependencies []string `yaml:"dependencies"`
}
