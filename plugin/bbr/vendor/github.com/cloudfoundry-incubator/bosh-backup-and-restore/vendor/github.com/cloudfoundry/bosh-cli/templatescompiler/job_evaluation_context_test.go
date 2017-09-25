package templatescompiler_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshreljob "github.com/cloudfoundry/bosh-cli/release/job"
	. "github.com/cloudfoundry/bosh-cli/release/resource"
	. "github.com/cloudfoundry/bosh-cli/templatescompiler"
	"github.com/cloudfoundry/bosh-cli/templatescompiler/erbrenderer"
	bierbrenderer "github.com/cloudfoundry/bosh-cli/templatescompiler/erbrenderer"
	"github.com/cloudfoundry/bosh-utils/errors"
)

var _ = Describe("JobEvaluationContext", func() {
	var (
		releaseJob              *boshreljob.Job
		jobProperties           *biproperty.Map
		instanceGroupProperties biproperty.Map
		deploymentProperties    biproperty.Map
		erbRenderer             erbrenderer.ERBRenderer
		jobEvaluationContext    bierbrenderer.TemplateEvaluationContext
		uuidGen                 *fakeuuid.FakeGenerator
	)

	BeforeEach(func() {
		releaseJob = boshreljob.NewJob(NewResource("fake-job-name", "", nil))
		releaseJob.Properties = map[string]boshreljob.PropertyDefinition{
			"property1.subproperty1": boshreljob.PropertyDefinition{
				Default: "spec-default",
			},
			"property2.subproperty2": boshreljob.PropertyDefinition{
				Default: "spec-default",
			},
		}

		deploymentProperties = biproperty.Map{}

		instanceGroupProperties = biproperty.Map{}

		uuidGen = fakeuuid.NewFakeGenerator()
		jobProperties = nil
	})

	JustBeforeEach(func() {
		logger := boshlog.NewLogger(boshlog.LevelNone)

		jobEvaluationContext = NewJobEvaluationContext(
			*releaseJob,
			jobProperties,
			instanceGroupProperties,
			deploymentProperties,
			"fake-deployment-name",
			"1.2.3.4",
			uuidGen,
			logger,
		)
	})

	act := func() RootContext {
		generatedJSON, err := jobEvaluationContext.MarshalJSON()
		Expect(err).ToNot(HaveOccurred())

		generatedContext := RootContext{}

		err = json.Unmarshal(generatedJSON, &generatedContext)
		Expect(err).ToNot(HaveOccurred())

		return generatedContext
	}

	It("it has a network context section with empty IP", func() {
		generatedContext := act()
		Expect(generatedContext.NetworkContexts["default"].IP).To(Equal(""))
	})

	It("it has address available in the spec", func() {
		generatedContext := act()
		Expect(generatedContext.Address).To(Equal("1.2.3.4"))
	})

	It("it has id available in the spec", func() {
		uuidGen.GeneratedUUID = "fake-uuid"
		generatedContext := act()
		Expect(generatedContext.ID).To(Equal("fake-uuid"))
	})

	It("it has az available in the spec", func() {
		generatedContext := act()
		Expect(generatedContext.AZ).To(Equal("unknown"))
	})

	It("it has bootstrap available in the spec", func() {
		generatedContext := act()
		Expect(generatedContext.Bootstrap).To(Equal(true))
	})
	Context("when the UUID generator raise an error", func() {
		It("it raises an error", func() {
			uuidGen.GenerateError = errors.Error("boom")
			_, err := jobEvaluationContext.MarshalJSON()
			Expect(err).To(HaveOccurred())
			Ω(err.Error()).Should(ContainSubstring("Setting job eval context's ID to UUID"))
		})
	})

	getValueFor := func(key string) string {
		logger := boshlog.NewLogger(boshlog.LevelNone)
		fs := boshsys.NewOsFileSystem(logger)
		commandRunner := boshsys.NewExecCmdRunner(logger)
		erbRenderer = erbrenderer.NewERBRenderer(fs, commandRunner, logger)

		srcFile, err := ioutil.TempFile("", "source.txt.erb")
		Expect(err).ToNot(HaveOccurred())
		defer os.Remove(srcFile.Name())

		erbContents := fmt.Sprintf("<%%= p('%s') %%>", key)
		_, err = srcFile.WriteString(erbContents)
		Expect(err).ToNot(HaveOccurred())

		destFile, err := fs.TempFile("dest.txt")
		Expect(err).ToNot(HaveOccurred())
		err = destFile.Close()
		Expect(err).ToNot(HaveOccurred())
		defer os.Remove(destFile.Name())

		jobEvaluationContext := NewJobEvaluationContext(
			*releaseJob,
			jobProperties,
			instanceGroupProperties,
			deploymentProperties,
			"fake-deployment-name",
			"1.2.3.4",
			uuidGen,
			logger,
		)

		err = erbRenderer.Render(srcFile.Name(), destFile.Name(), jobEvaluationContext)
		Expect(err).ToNot(HaveOccurred())
		contents, err := ioutil.ReadFile(destFile.Name())
		Expect(err).ToNot(HaveOccurred())
		return (string)(contents)
	}

	Context("when a deployment and instance group set a property", func() {
		BeforeEach(func() {
			deploymentProperties = biproperty.Map{
				"property1": biproperty.Map{
					"subproperty1": "value-from-global-properties",
				},
			}

			instanceGroupProperties = biproperty.Map{
				"property1": biproperty.Map{
					"subproperty1": "value-from-cluster-properties",
				},
			}
		})

		It("gives precedence to the instance group value", func() {
			Expect(getValueFor("property1.subproperty1")).To(Equal("value-from-cluster-properties"))
		})
	})

	Context("when a deployment sets a property", func() {
		BeforeEach(func() {
			deploymentProperties = biproperty.Map{
				"property1": biproperty.Map{
					"subproperty1": "value-from-global-properties",
				},
			}
		})

		It("uses the value", func() {
			Expect(getValueFor("property1.subproperty1")).To(Equal("value-from-global-properties"))
		})
	})

	Context("when an instance group sets a property", func() {
		BeforeEach(func() {
			instanceGroupProperties = biproperty.Map{
				"property1": biproperty.Map{
					"subproperty1": "value-from-cluster-properties",
				},
			}
		})

		It("uses the value", func() {
			Expect(getValueFor("property1.subproperty1")).To(Equal("value-from-cluster-properties"))
		})
	})

	Context("when a property is not set", func() {
		It("uses the release's default value", func() {
			Expect(getValueFor("property1.subproperty1")).To(Equal("spec-default"))
		})
	})

	Context("when a job sets a property", func() {
		BeforeEach(func() {
			jobProperties = &biproperty.Map{
				"property1": biproperty.Map{
					"subproperty1": "job-property",
				},
			}
		})

		It("uses the value", func() {
			Expect(getValueFor("property1.subproperty1")).To(Equal("job-property"))
		})

		Context("when the instance group also sets a property", func() {
			instanceGroupProperties = biproperty.Map{
				"property2": biproperty.Map{
					"subproperty2": "instance-group-property",
				},
			}

			It("is not used", func() {
				Expect(getValueFor("property2.subproperty2")).To(Equal("spec-default"))
			})
		})
	})

	Context("when the job sets a property to an empty hash ({})", func() {
		BeforeEach(func() {
			jobProperties = &biproperty.Map{}
		})

		Context("when an instance group sets a property", func() {
			BeforeEach(func() {
				instanceGroupProperties = biproperty.Map{
					"property1": biproperty.Map{
						"subproperty1": "value-from-instance-group-properties",
					},
				}
			})

			It("does not use the instance group value", func() {
				Expect(getValueFor("property1.subproperty1")).To(Equal("spec-default"))
			})
		})

		Context("when an deployment sets a property", func() {
			BeforeEach(func() {
				deploymentProperties = biproperty.Map{
					"property1": biproperty.Map{
						"subproperty1": "value-from-global-properties",
					},
				}
			})

			It("does not use the instance group value", func() {
				Expect(getValueFor("property1.subproperty1")).To(Equal("spec-default"))
			})
		})
	})
})
