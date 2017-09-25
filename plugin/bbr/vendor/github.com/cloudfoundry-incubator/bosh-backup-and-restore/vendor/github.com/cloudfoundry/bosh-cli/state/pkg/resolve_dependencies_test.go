package pkg_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshrelpkg "github.com/cloudfoundry/bosh-cli/release/pkg"
	. "github.com/cloudfoundry/bosh-cli/state/pkg"
)

var _ = Describe("DependencyResolver", func() {
	It("supports a single dependency", func() {
		a := newPkg("a", "", []string{"b"})
		b := newPkg("b", "", nil)
		a.AttachDependencies([]*boshrelpkg.Package{b})

		deps := ResolveDependencies(a)
		Expect(deps).To(Equal([]boshrelpkg.Compilable{b}))
	})

	It("supports a transitive dependency", func() {
		a := newPkg("a", "", []string{"b"})
		b := newPkg("b", "", []string{"c"})
		c := newPkg("c", "", nil)
		a.AttachDependencies([]*boshrelpkg.Package{b})
		b.AttachDependencies([]*boshrelpkg.Package{c})

		deps := ResolveDependencies(a)
		Expect(deps).To(Equal([]boshrelpkg.Compilable{c, b}))
	})

	It("supports simple cycles", func() {
		a := newPkg("a", "", []string{"b"})
		b := newPkg("b", "", []string{"a"})
		a.AttachDependencies([]*boshrelpkg.Package{b})
		b.AttachDependencies([]*boshrelpkg.Package{a})

		deps := ResolveDependencies(a)
		Expect(deps).ToNot(ContainElement(a))
		Expect(deps).To(Equal([]boshrelpkg.Compilable{b}))
	})

	It("supports triangular cycles", func() {
		a := newPkg("a", "", []string{"b"})
		b := newPkg("b", "", []string{"c"})
		c := newPkg("c", "", []string{"a"})
		a.AttachDependencies([]*boshrelpkg.Package{b})
		b.AttachDependencies([]*boshrelpkg.Package{c})
		c.AttachDependencies([]*boshrelpkg.Package{a})

		deps := ResolveDependencies(a)
		Expect(deps).ToNot(ContainElement(a))
		Expect(deps).To(Equal([]boshrelpkg.Compilable{c, b}))
	})

	It("supports no cycles", func() {
		a := newPkg("a", "", []string{"b", "c"})
		b := newPkg("b", "", nil)
		c := newPkg("c", "", []string{"b"})
		a.AttachDependencies([]*boshrelpkg.Package{b, c})
		c.AttachDependencies([]*boshrelpkg.Package{b})

		deps := ResolveDependencies(a)
		Expect(deps).ToNot(ContainElement(a))
		Expect(deps).To(Equal([]boshrelpkg.Compilable{c, b}))
	})

	It("supports diamond cycles", func() {
		a := newPkg("a", "", []string{"c"})
		b := newPkg("b", "", []string{"a"})
		c := newPkg("c", "", []string{"d"})
		d := newPkg("d", "", []string{"b"})
		a.AttachDependencies([]*boshrelpkg.Package{c})
		b.AttachDependencies([]*boshrelpkg.Package{a})
		c.AttachDependencies([]*boshrelpkg.Package{d})
		d.AttachDependencies([]*boshrelpkg.Package{b})

		deps := ResolveDependencies(a)
		Expect(deps).ToNot(ContainElement(a))
		Expect(deps).To(Equal([]boshrelpkg.Compilable{b, d, c}))
	})

	It("supports sibling dependencies", func() {
		a := newPkg("a", "", []string{"b", "c"})
		b := newPkg("b", "", []string{"c", "d"})
		c := newPkg("c", "", []string{"d"})
		d := newPkg("d", "", nil)
		a.AttachDependencies([]*boshrelpkg.Package{b, c})
		b.AttachDependencies([]*boshrelpkg.Package{c, d})
		c.AttachDependencies([]*boshrelpkg.Package{d})

		deps := ResolveDependencies(a)
		Expect(deps).ToNot(ContainElement(a))
		Expect(deps).To(Equal([]boshrelpkg.Compilable{d, c, b}))
	})
})
