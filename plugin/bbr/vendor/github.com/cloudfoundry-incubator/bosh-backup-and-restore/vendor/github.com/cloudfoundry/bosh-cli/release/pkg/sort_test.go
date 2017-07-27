package pkg_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	gomegafmt "github.com/onsi/gomega/format"

	. "github.com/cloudfoundry/bosh-cli/release/pkg"
	. "github.com/cloudfoundry/bosh-cli/release/resource"
)

var _ = Describe("Sort", func() {
	var (
		pkg1, pkg2 *Package
		pkgs       []Compilable
	)

	gomegafmt.UseStringerRepresentation = true

	var indexOf = func(pkgs []Compilable, pkg Compilable) int {
		for index, currentPkg := range pkgs {
			if currentPkg == pkg {
				return index
			}
		}
		return -1
	}

	var expectSorted = func(sortedPackages []Compilable) {
		for _, pkg := range pkgs {
			sortedIndex := indexOf(sortedPackages, pkg)
			for _, dependencyPkg := range pkg.Deps() {
				errorMessage := fmt.Sprintf("Package '%s' should be compiled after package '%s'", pkg.Name(), dependencyPkg.Name())
				Expect(sortedIndex).To(BeNumerically(">", indexOf(sortedPackages, dependencyPkg)), errorMessage)
			}
		}
	}

	newPkg := func(name string) *Package {
		return NewPackage(NewResource(name, "", nil), nil)
	}

	BeforeEach(func() {
		pkg1 = newPkg("fake-pkg-1")
		pkg2 = newPkg("fake-pkg-2")
		pkgs = []Compilable{pkg1, pkg2}
	})

	Context("disjoint pkgs have a valid compilation sequence", func() {
		It("returns an ordered set of package compilation", func() {
			sortedPackages, err := Sort(pkgs)
			Expect(err).ToNot(HaveOccurred())
			Expect(sortedPackages).To(ContainElement(pkg1))
			Expect(sortedPackages).To(ContainElement(pkg2))
		})
	})

	Context("dependent pkgs", func() {
		BeforeEach(func() {
			pkg1.Dependencies = []*Package{pkg2}
		})

		It("returns an ordered set of package compilation", func() {
			sortedPackages, err := Sort(pkgs)
			Expect(err).ToNot(HaveOccurred())
			Expect(sortedPackages).To(Equal([]Compilable{pkg2, pkg1}))
		})
	})

	Context("complex graph of dependent pkgs", func() {
		var (
			package3, package4 *Package
		)

		BeforeEach(func() {
			package3 = newPkg("fake-pkg-3")
			pkg1.Dependencies = []*Package{pkg2, package3}
			package4 = newPkg("fake-pkg-4")
			package4.Dependencies = []*Package{package3, pkg2}
			pkgs = []Compilable{pkg1, pkg2, package3, package4}
		})

		It("returns an ordered set of package compilation", func() {
			sortedPackages, err := Sort(pkgs)
			Expect(err).ToNot(HaveOccurred())
			expectSorted(sortedPackages)
		})
	})

	Context("cicular dependencies", func() {
		var (
			pkg3 *Package
		)

		BeforeEach(func() {
			pkg1 = newPkg("pkg1-name")
			pkg2 = newPkg("pkg2-name")
			pkg3 = newPkg("pkg3-name")
			pkg1.Dependencies = []*Package{pkg3}
			pkg2.Dependencies = []*Package{pkg1}
			pkg3.Dependencies = []*Package{pkg2}
			pkgs = []Compilable{pkg1, pkg2, pkg3}
		})

		It("returns an error", func() {
			_, err := Sort(pkgs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Circular dependency detected while sorting packages"))
		})
	})

	Context("graph with transitively dependent pkgs", func() {
		var (
			package3, package4, package5 *Package
		)

		BeforeEach(func() {
			pkg2.Dependencies = []*Package{pkg1}
			package3 = newPkg("fake-pkg-3")
			package3.Dependencies = []*Package{pkg2}
			package4 = newPkg("fake-pkg-4")
			package5 = newPkg("fake-pkg-5")
			package5.Dependencies = []*Package{pkg2}
			pkgs = []Compilable{pkg1, pkg2, package3, package4, package5}
		})

		It("returns an ordered set of package compilation", func() {
			sortedPackages, err := Sort(pkgs)
			Expect(err).ToNot(HaveOccurred())
			expectSorted(sortedPackages)
		})
	})

	Context("graph from a bosh-release (real life example)", func() {
		BeforeEach(func() {
			nginx := newPkg("nginx")
			genisoimage := newPkg("genisoimage")
			powerdns := newPkg("powerdns")
			ruby := newPkg("ruby")

			blobstore := newPkg("blobstore")
			blobstore.Dependencies = []*Package{ruby}

			mysql := newPkg("mysql")

			nats := newPkg("nats")
			nats.Dependencies = []*Package{ruby}

			common := newPkg("common")
			redis := newPkg("redis")
			libpq := newPkg("libpq")
			postgres := newPkg("postgres")

			registry := newPkg("registry")
			registry.Dependencies = []*Package{libpq, mysql, ruby}

			director := newPkg("director")
			director.Dependencies = []*Package{libpq, mysql, ruby}

			healthMonitor := newPkg("health_monitor")
			healthMonitor.Dependencies = []*Package{ruby}

			pkgs = []Compilable{
				nginx,
				genisoimage,
				powerdns,
				blobstore, // before ruby
				ruby,
				mysql,
				nats,
				common,
				director, // before libpq, postgres; after ruby
				redis,
				registry, // before libpq, postgres; after ruby
				libpq,
				postgres,
				healthMonitor, // after ruby, libpq, postgres
			}
		})

		It("orders bosh-release pkgs for compilation", func() {
			sortedPackages, err := Sort(pkgs)
			Expect(err).ToNot(HaveOccurred())
			expectSorted(sortedPackages)
		})
	})

	Context("graph with sibling dependencies (real life example)", func() {
		var (
			golang, runC, garden, guardian *Package
		)

		BeforeEach(func() {
			golang = newPkg("golang")

			runC = newPkg("runC")
			runC.Dependencies = []*Package{golang}

			guardian = newPkg("guardian")
			guardian.Dependencies = []*Package{runC, golang}

			garden = newPkg("garden")
			garden.Dependencies = []*Package{guardian}

			pkgs = []Compilable{guardian, garden, runC, golang}
		})

		It("orders the packages as: golang, runC, guardian, garden", func() {
			sortedPackages, err := Sort(pkgs)
			Expect(err).ToNot(HaveOccurred())
			Expect(sortedPackages[0].Name()).To(Equal(golang.Name()))
			Expect(sortedPackages[1].Name()).To(Equal(runC.Name()))
			Expect(sortedPackages[2].Name()).To(Equal(guardian.Name()))
			Expect(sortedPackages[3].Name()).To(Equal(garden.Name()))
		})
	})

	Context("graph with circular dependency", func() {
		var (
			package1, package2, package3 *Package
		)

		BeforeEach(func() {
			package1 = newPkg("fake-package-name-1")

			package2 = newPkg("fake-package-name-2")
			package2.Dependencies = []*Package{package1}

			package3 = newPkg("fake-package-name-3")
			package3.Dependencies = []*Package{package2}

			package1.Dependencies = []*Package{package3}
		})

		It("returns an error", func() {
			_, err := Sort([]Compilable{package1, package2, package3})
			Expect(err).NotTo(BeNil())
		})
	})

	Context("graph from a CPI release", func() {
		BeforeEach(func() {
			pid_utils := newPkg("pid_utils")

			iptables := newPkg("iptables")

			bosh_io_release_resource := newPkg("bosh_io_release_resource")

			s3_resource := newPkg("s3_resource")

			bosh_io_stemcell_resource := newPkg("bosh_io_stemcell_resource")

			golang := newPkg("golang")

			baggageclaim := newPkg("baggageclaim")
			baggageclaim.Dependencies = []*Package{golang}

			jettison := newPkg("jettison")
			jettison.Dependencies = []*Package{golang}

			cf_resource := newPkg("cf_resource")

			golang_161 := newPkg("golang_1.6.1")

			runc := newPkg("runc")
			runc.Dependencies = []*Package{golang_161}

			guardian := newPkg("guardian")
			guardian.Dependencies = []*Package{golang_161, runc}

			btrfs_tools := newPkg("btrfs_tools")

			docker_image_resource := newPkg("docker_image_resource")

			resource_discovery := newPkg("resource_discovery")
			resource_discovery.Dependencies = []*Package{golang}

			github_release_resource := newPkg("github_release_resource")

			shadow := newPkg("shadow")

			vagrant_cloud_resource := newPkg("vagrant_cloud_resource")

			pool_resource := newPkg("pool_resource")

			bosh_deployment_resource := newPkg("bosh_deployment_resource")

			generated_worker_key := newPkg("generated_worker_key")

			archive_resource := newPkg("archive_resource")

			time_resource := newPkg("time_resource")

			git_resource := newPkg("git_resource")

			busybox := newPkg("busybox")

			semver_resource := newPkg("semver_resource")

			hg_resource := newPkg("hg_resource")

			tar := newPkg("tar")

			tracker_resource := newPkg("tracker_resource")

			pkgs = []Compilable{
				bosh_io_release_resource,
				guardian,
				iptables,
				pid_utils,
				jettison,
				docker_image_resource,
				github_release_resource,
				vagrant_cloud_resource,
				golang,
				generated_worker_key,
				golang_161,
				git_resource,
				semver_resource,
				tar,
				tracker_resource,
				shadow,
				cf_resource,
				pool_resource,
				baggageclaim,
				bosh_deployment_resource,
				bosh_io_stemcell_resource,
				archive_resource,
				s3_resource,
				time_resource,
				btrfs_tools,
				busybox,
				resource_discovery,
				hg_resource,
				runc,
			}
		})

		It("sorts the packages correctly", func() {
			hasBeenLoaded := map[string]bool{}

			for _, pkg := range pkgs {
				hasBeenLoaded[pkg.Name()] = false
			}

			sortedPackages, err := Sort(pkgs)
			Expect(err).ToNot(HaveOccurred())

			for _, pkg := range sortedPackages {
				if pkg.Deps() != nil {
					for _, dep := range pkg.Deps() {
						Expect(hasBeenLoaded[dep.Name()]).To(BeTrue())
					}
				}
				hasBeenLoaded[pkg.Name()] = true
			}
		})
	})
})
