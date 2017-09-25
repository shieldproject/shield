package models

type Package struct {
	Name    string
	Version string
	Source  Source
}

func (s Package) BundleName() string {
	return s.Name
}

func (s Package) BundleVersion() string {
	if len(s.Version) == 0 {
		panic("Internal inconsistency: Expected package.Version to be non-empty")
	}
	return s.Version + "-" + s.Source.Sha1.String()
}

type LocalPackage struct {
	Name    string
	Version string
}

func (s LocalPackage) BundleName() string {
	return s.Name
}

func (s LocalPackage) BundleVersion() string {
	if len(s.Version) == 0 {
		panic("Internal inconsistency: Expected localPackage.Version to be non-empty")
	}
	return s.Version
}
