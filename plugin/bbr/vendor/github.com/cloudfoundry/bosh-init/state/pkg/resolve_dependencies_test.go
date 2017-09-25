package pkg_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-init/state/pkg"

	birelpkg "github.com/cloudfoundry/bosh-init/release/pkg"
)

var _ = Describe("DependencyResolver", func() {
	It("supports a single dependency", func() {
		a := birelpkg.Package{Name: "a"}
		b := birelpkg.Package{Name: "b"}
		a.Dependencies = []*birelpkg.Package{&b}

		deps := ResolveDependencies(&a)
		Expect(deps).To(Equal([]*birelpkg.Package{&b}))
	})

	It("supports a transitive dependency", func() {
		a := birelpkg.Package{Name: "a"}
		b := birelpkg.Package{Name: "b"}
		a.Dependencies = []*birelpkg.Package{&b}
		c := birelpkg.Package{Name: "c"}
		b.Dependencies = []*birelpkg.Package{&c}

		deps := ResolveDependencies(&a)
		Expect(deps).To(Equal([]*birelpkg.Package{&c, &b}))
	})

	It("supports simple cycles", func() {
		a := birelpkg.Package{Name: "a"}
		b := birelpkg.Package{Name: "b"}
		a.Dependencies = []*birelpkg.Package{&b}
		b.Dependencies = []*birelpkg.Package{&a}

		deps := ResolveDependencies(&a)
		Expect(deps).ToNot(ContainElement(&a))
		Expect(deps).To(Equal([]*birelpkg.Package{&b}))
	})

	It("supports triangular cycles", func() {
		a := birelpkg.Package{Name: "a"}
		b := birelpkg.Package{Name: "b"}
		a.Dependencies = []*birelpkg.Package{&b}
		c := birelpkg.Package{Name: "c"}
		b.Dependencies = []*birelpkg.Package{&c}
		c.Dependencies = []*birelpkg.Package{&a}

		deps := ResolveDependencies(&a)
		Expect(deps).ToNot(ContainElement(&a))
		Expect(deps).To(Equal([]*birelpkg.Package{&c, &b}))
	})

	It("supports no cycles", func() {
		a := birelpkg.Package{Name: "a"}
		b := birelpkg.Package{Name: "b"}
		c := birelpkg.Package{Name: "c"}
		a.Dependencies = []*birelpkg.Package{&b, &c}
		c.Dependencies = []*birelpkg.Package{&b}

		deps := ResolveDependencies(&a)
		Expect(deps).ToNot(ContainElement(&a))
		Expect(deps).To(Equal([]*birelpkg.Package{&c, &b}))
	})

	It("supports diamond cycles", func() {
		a := birelpkg.Package{Name: "a"}
		b := birelpkg.Package{Name: "b"}
		c := birelpkg.Package{Name: "c"}
		d := birelpkg.Package{Name: "d"}

		a.Dependencies = []*birelpkg.Package{&c}
		b.Dependencies = []*birelpkg.Package{&a}
		c.Dependencies = []*birelpkg.Package{&d}
		d.Dependencies = []*birelpkg.Package{&b}

		deps := ResolveDependencies(&a)
		Expect(deps).ToNot(ContainElement(&a))
		Expect(deps).To(Equal([]*birelpkg.Package{&b, &d, &c}))
	})

	It("supports sibling dependencies", func() {
		a := birelpkg.Package{Name: "a"}
		b := birelpkg.Package{Name: "b"}
		c := birelpkg.Package{Name: "c"}
		d := birelpkg.Package{Name: "d"}

		a.Dependencies = []*birelpkg.Package{&b, &c}
		b.Dependencies = []*birelpkg.Package{&c, &d}
		c.Dependencies = []*birelpkg.Package{&d}

		deps := ResolveDependencies(&a)
		Expect(deps).ToNot(ContainElement(&a))
		Expect(deps).To(Equal([]*birelpkg.Package{&d, &c, &b}))
	})
})
