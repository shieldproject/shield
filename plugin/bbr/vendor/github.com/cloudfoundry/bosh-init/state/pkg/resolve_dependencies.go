package pkg

import (
	birelpkg "github.com/cloudfoundry/bosh-init/release/pkg"
)

func ResolveDependencies(pkg *birelpkg.Package) []*birelpkg.Package {
	return reverse(resolveInner(pkg, []*birelpkg.Package{}))
}

func resolveInner(pkg *birelpkg.Package, noFollow []*birelpkg.Package) []*birelpkg.Package {
	all := []*birelpkg.Package{}
	for _, depPkg := range pkg.Dependencies {
		if !contains(all, depPkg) && !contains(noFollow, depPkg) {
			all = append(all, depPkg)

			tDeps := resolveInner(depPkg, joinUnique(all, noFollow))
			for _, tDepPkg := range tDeps {
				all = append(all, tDepPkg)
			}
		}
	}

	for i, el := range all {
		if el == pkg {
			all = append(all[:i], all[i+1:]...)
		}
	}
	return all
}

func contains(list []*birelpkg.Package, element *birelpkg.Package) bool {
	for _, pkg := range list {
		if element == pkg {
			return true
		}
	}
	return false
}

func joinUnique(a []*birelpkg.Package, b []*birelpkg.Package) []*birelpkg.Package {
	joined := []*birelpkg.Package{}
	joined = append(joined, a...)
	for _, pkg := range b {
		if !contains(a, pkg) {
			joined = append(joined, pkg)
		}
	}
	return joined
}

func reverse(a []*birelpkg.Package) []*birelpkg.Package {
	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}

	return a
}
