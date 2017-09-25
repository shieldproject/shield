package pkg_test

import (
	"fmt"

	. "github.com/cloudfoundry/bosh-init/release/pkg"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	gomegafmt "github.com/onsi/gomega/format"
)

var _ = Describe("Sort", func() {
	var (
		packages []*Package
	)

	gomegafmt.UseStringerRepresentation = true

	var indexOf = func(packages []*Package, pkg *Package) int {
		for index, currentPkg := range packages {
			if currentPkg == pkg {
				return index
			}
		}
		return -1
	}

	var expectSorted = func(sortedPackages []*Package) {
		for _, pkg := range packages {
			sortedIndex := indexOf(sortedPackages, pkg)
			for _, dependencyPkg := range pkg.Dependencies {
				errorMessage := fmt.Sprintf("Package '%s' should be compiled after package '%s'", pkg.Name, dependencyPkg.Name)
				Expect(sortedIndex).To(BeNumerically(">", indexOf(sortedPackages, dependencyPkg)), errorMessage)
			}
		}
	}

	var package1, package2 Package

	BeforeEach(func() {
		package1 = Package{
			Name: "fake-package-name-1",
		}
		package2 = Package{
			Name: "fake-package-name-2",
		}
		packages = []*Package{&package1, &package2}
	})

	Context("disjoint packages have a valid compilation sequence", func() {
		It("returns an ordered set of package compilation", func() {
			sortedPackages, _ := Sort(packages)

			Expect(sortedPackages).To(ContainElement(&package1))
			Expect(sortedPackages).To(ContainElement(&package2))
		})
	})

	Context("dependent packages", func() {
		BeforeEach(func() {
			package1.Dependencies = []*Package{&package2}
		})

		It("returns an ordered set of package compilation", func() {
			sortedPackages, _ := Sort(packages)

			Expect(sortedPackages).To(Equal([]*Package{&package2, &package1}))
		})
	})

	Context("complex graph of dependent packages", func() {
		var package3, package4 Package

		BeforeEach(func() {
			package1.Dependencies = []*Package{&package2, &package3}
			package3 = Package{
				Name: "fake-package-name-3",
			}
			package4 = Package{
				Name:         "fake-package-name-4",
				Dependencies: []*Package{&package3, &package2},
			}
			packages = []*Package{&package1, &package2, &package3, &package4}
		})

		It("returns an ordered set of package compilation", func() {
			sortedPackages, _ := Sort(packages)

			expectSorted(sortedPackages)
		})
	})

	Context("graph with transitively dependent packages", func() {
		var package3, package4, package5 Package

		BeforeEach(func() {
			package3 = Package{
				Name: "fake-package-name-3",
			}
			package4 = Package{
				Name: "fake-package-name-4",
			}
			package5 = Package{
				Name: "fake-package-name-5",
			}

			package3.Dependencies = []*Package{&package2}
			package2.Dependencies = []*Package{&package1}

			package5.Dependencies = []*Package{&package2}

			packages = []*Package{&package1, &package2, &package3, &package4, &package5}
		})

		It("returns an ordered set of package compilation", func() {
			sortedPackages, _ := Sort(packages)

			expectSorted(sortedPackages)
		})
	})

	Context("graph from a BOSH release", func() {
		BeforeEach(func() {
			nginx := Package{Name: "nginx"}
			genisoimage := Package{Name: "genisoimage"}
			powerdns := Package{Name: "powerdns"}
			ruby := Package{Name: "ruby"}

			blobstore := Package{
				Name:         "blobstore",
				Dependencies: []*Package{&ruby},
			}

			mysql := Package{Name: "mysql"}

			nats := Package{
				Name:         "nats",
				Dependencies: []*Package{&ruby},
			}

			common := Package{Name: "common"}
			redis := Package{Name: "redis"}
			libpq := Package{Name: "libpq"}
			postgres := Package{Name: "postgres"}

			registry := Package{
				Name:         "registry",
				Dependencies: []*Package{&libpq, &mysql, &ruby},
			}

			director := Package{
				Name:         "director",
				Dependencies: []*Package{&libpq, &mysql, &ruby},
			}

			healthMonitor := Package{
				Name:         "health_monitor",
				Dependencies: []*Package{&ruby},
			}

			packages = []*Package{
				&nginx,
				&genisoimage,
				&powerdns,
				&blobstore, // before ruby
				&ruby,
				&mysql,
				&nats,
				&common,
				&director, // before libpq, postgres; after ruby
				&redis,
				&registry, // before libpq, postgres; after ruby
				&libpq,
				&postgres,
				&healthMonitor, // after ruby, libpq, postgres
			}
		})

		It("orders BOSH release packages for compilation (example)", func() {
			sortedPackages, _ := Sort(packages)

			expectSorted(sortedPackages)
		})
	})

	Context("graph with sibling dependencies", func() {
		var (
			golang, runC, garden, guardian Package
		)

		BeforeEach(func() {
			golang = Package{Name: "golang"}

			runC = Package{
				Name:         "runC",
				Dependencies: []*Package{&golang},
			}

			guardian = Package{
				Name:         "guardian",
				Dependencies: []*Package{&runC, &golang},
			}

			garden = Package{
				Name:         "garden",
				Dependencies: []*Package{&guardian},
			}

			packages = []*Package{
				&guardian,
				&garden,
				&runC,
				&golang,
			}
		})

		It("orders the packages as: golang, runC, guardian, garden", func() {
			sortedPackages, _ := Sort(packages)

			Expect(sortedPackages[0].Name).To(Equal(golang.Name))
			Expect(sortedPackages[1].Name).To(Equal(runC.Name))
			Expect(sortedPackages[2].Name).To(Equal(guardian.Name))
			Expect(sortedPackages[3].Name).To(Equal(garden.Name))
		})
	})

	Context("Graph with circular dependency", func() {
		var (
			package1,
			package2,
			package3 Package
		)
		BeforeEach(func() {
			package1 = Package{
				Name:         "fake-package-name-1",
				Dependencies: []*Package{},
			}
			package2 = Package{
				Name:         "fake-package-name-2",
				Dependencies: []*Package{&package1},
			}
			package3 = Package{
				Name:         "fake-package-name-3",
				Dependencies: []*Package{&package2},
			}

			package1.Dependencies = append(package1.Dependencies, &package3)
		})
		It("returns an error", func() {
			packages := []*Package{
				&package1,
				&package2,
				&package3,
			}
			_, err := Sort(packages)
			Expect(err).NotTo(BeNil())

		})
	})

	Context("Graph from a CPI release", func() {
		var packages []*Package
		BeforeEach(func() {
			pid_utils := Package{Name: "pid_utils"}

			iptables := Package{Name: "iptables"}

			bosh_io_release_resource := Package{Name: "bosh_io_release_resource"}

			s3_resource := Package{Name: "s3_resource"}

			bosh_io_stemcell_resource := Package{Name: "bosh_io_stemcell_resource"}

			golang := Package{Name: "golang"}

			baggageclaim := Package{
				Name:         "baggageclaim",
				Dependencies: []*Package{&golang},
			}

			jettison := Package{
				Name:         "jettison",
				Dependencies: []*Package{&golang},
			}
			cf_resource := Package{Name: "cf_resource"}

			golang_161 := Package{Name: "golang_1.6.1"}

			runc := Package{
				Name:         "runc",
				Dependencies: []*Package{&golang_161},
			}

			guardian := Package{
				Name:         "guardian",
				Dependencies: []*Package{&golang_161, &runc},
			}

			btrfs_tools := Package{Name: "btrfs_tools"}

			docker_image_resource := Package{Name: "docker_image_resource"}

			resource_discovery := Package{
				Name:         "resource_discovery",
				Dependencies: []*Package{&golang},
			}

			github_release_resource := Package{Name: "github_release_resource"}

			shadow := Package{Name: "shadow"}

			vagrant_cloud_resource := Package{Name: "vagrant_cloud_resource"}

			pool_resource := Package{Name: "pool_resource"}

			bosh_deployment_resource := Package{Name: "bosh_deployment_resource"}

			generated_worker_key := Package{Name: "generated_worker_key"}

			archive_resource := Package{Name: "archive_resource"}

			time_resource := Package{Name: "time_resource"}

			git_resource := Package{Name: "git_resource"}

			busybox := Package{Name: "busybox"}

			semver_resource := Package{Name: "semver_resource"}

			hg_resource := Package{Name: "hg_resource"}

			tar := Package{Name: "tar"}

			tracker_resource := Package{Name: "tracker_resource"}

			packages = []*Package{
				&bosh_io_release_resource,
				&guardian,
				&iptables,
				&pid_utils,
				&jettison,
				&docker_image_resource,
				&github_release_resource,
				&vagrant_cloud_resource,
				&golang,
				&generated_worker_key,
				&golang_161,
				&git_resource,
				&semver_resource,
				&tar,
				&tracker_resource,
				&shadow,
				&cf_resource,
				&pool_resource,
				&baggageclaim,
				&bosh_deployment_resource,
				&bosh_io_stemcell_resource,
				&archive_resource,
				&s3_resource,
				&time_resource,
				&btrfs_tools,
				&busybox,
				&resource_discovery,
				&hg_resource,
				&runc}
		})

		It("should sort the packages correctly", func() {
			hasBeenLoaded := map[string]bool{}

			for _, pkg := range packages {
				hasBeenLoaded[pkg.Name] = false
			}

			sortedPackages, _ := Sort(packages)

			for _, pkg := range sortedPackages {
				if pkg.Dependencies != nil {
					for _, dep := range pkg.Dependencies {
						Expect(hasBeenLoaded[dep.Name]).To(BeTrue())
					}
				}
				hasBeenLoaded[pkg.Name] = true
			}
		})
	})
})
