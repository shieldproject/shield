package pkg

type Package struct {
	Name          string
	Fingerprint   string
	SHA1          string
	Stemcell      string
	Dependencies  []*Package
	ExtractedPath string
	ArchivePath   string
}

func (p Package) String() string {
	return p.Name
}
