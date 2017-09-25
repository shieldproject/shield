package pkg

import (
	"errors"
	"fmt"
)

// Topologically sorts an array of packages
func Sort(releasePackages []*Package) ([]*Package, error) {
	sortedPackages := []*Package{}

	incomingEdges, outgoingEdges := getEdgeMaps(releasePackages)
	noIncomingEdgesSet := []*Package{}

	for pkg, edgeList := range incomingEdges {
		if len(edgeList) == 0 {
			noIncomingEdgesSet = append(noIncomingEdgesSet, pkg)
		}
	}
	for len(noIncomingEdgesSet) > 0 {
		elem := noIncomingEdgesSet[0]
		noIncomingEdgesSet = noIncomingEdgesSet[1:]

		sortedPackages = append([]*Package{elem}, sortedPackages...)

		for _, pkg := range outgoingEdges[elem] {
			incomingEdges[pkg] = removeFromList(incomingEdges[pkg], elem)
			if len(incomingEdges[pkg]) == 0 {
				noIncomingEdgesSet = append(noIncomingEdgesSet, pkg)
			}
		}
	}
	for _, edges := range incomingEdges {
		if len(edges) > 0 {
			return nil, errors.New("Circular dependency detected while sorting packages")
		}
	}
	return sortedPackages, nil
}

func removeFromList(packageList []*Package, pkg *Package) []*Package {
	for idx, elem := range packageList {
		if elem == pkg {
			return append(packageList[:idx], packageList[idx+1:]...)
		}
	}
	panic(fmt.Sprintf("Expected %s to be in dependency graph", pkg.Name))
}

func getEdgeMaps(releasePackages []*Package) (map[*Package][]*Package, map[*Package][]*Package) {
	incomingEdges := make(map[*Package][]*Package)
	outgoingEdges := make(map[*Package][]*Package)

	for _, pkg := range releasePackages {
		incomingEdges[pkg] = []*Package{}
	}

	for _, pkg := range releasePackages {
		if pkg.Dependencies != nil {
			for _, dep := range pkg.Dependencies {
				incomingEdges[dep] = append(incomingEdges[dep], pkg)
				outgoingEdges[pkg] = append(outgoingEdges[pkg], dep)
			}
		}
	}
	return incomingEdges, outgoingEdges
}
