package pkg

type PackageRepo struct {
	repo map[string]*Package
}

func (pr *PackageRepo) FindOrCreatePackage(pkgName string) *Package {
	if pr.repo == nil {
		pr.repo = make(map[string]*Package)
	}

	pkg, ok := pr.repo[pkgName]
	if ok {
		return pkg
	}
	newPackage := &Package{
		Name: pkgName,
	}

	pr.repo[pkgName] = newPackage

	return newPackage
}
