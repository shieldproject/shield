package template_test

import (
	"errors"

	"github.com/cppforlife/go-patch/patch"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/deployment/template"
	boshtpl "github.com/cloudfoundry/bosh-cli/director/template"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("TemplateFactory", func() {
	var (
		fileTemplatePath string
		fakeFs           *fakesys.FakeFileSystem
		templateFactory  DeploymentTemplateFactory
	)

	BeforeEach(func() {
		fileTemplatePath = "fake-deployment-path"
		fakeFs = fakesys.NewFakeFileSystem()
		templateFactory = NewDeploymentTemplateFactory(fakeFs)
	})

	Context("NewTemplateFromPath", func() {
		Context("when the path does not exist", func() {
			BeforeEach(func() {
				err := fakeFs.RemoveAll(fileTemplatePath)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				_, err := templateFactory.NewDeploymentTemplateFromPath(fileTemplatePath)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when a file read error occurs", func() {
			BeforeEach(func() {
				fakeFs.ReadFileError = errors.New("fake-read-file-error")
			})

			It("returns an error", func() {
				_, err := templateFactory.NewDeploymentTemplateFromPath(fileTemplatePath)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when creation and interpolation succeeds", func() {
			It("interpolates variables and later resolves their values", func() {
				path := "/path/to/fake-deployment-yml"
				fakeFs.WriteFileString(path, `---
name: fake-deployment-manifest
resource_pools:
- name: fake-resource-pool-name
  stemcell:
    url: ((url))
`)

				template, err := templateFactory.NewDeploymentTemplateFromPath(path)
				Expect(err).ToNot(HaveOccurred())

				vars := boshtpl.StaticVariables{"url": "file://stemcell.tgz"}
				ops := patch.Ops{
					patch.ReplaceOp{Path: patch.MustNewPointerFromString("/name"), Value: "replaced-name"},
				}

				interpolatedTemplate, err := template.Evaluate(vars, ops)
				Expect(err).ToNot(HaveOccurred())

				Expect(string(interpolatedTemplate.Content())).To(Equal(`name: replaced-name
resource_pools:
- name: fake-resource-pool-name
  stemcell:
    url: file://stemcell.tgz
`))
			})
		})
	})
})
