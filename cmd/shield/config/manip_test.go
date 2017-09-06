package config_test

import (
	"fmt"
	"sort"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/starkandwayne/shield/api"
	. "github.com/starkandwayne/shield/cmd/shield/config"
)

func backendAtIndex(index int) *api.Backend {
	return &api.Backend{
		Name:              fmt.Sprintf("backend%d", index+1),
		Address:           fmt.Sprintf("http://addr%d", index+1),
		Token:             fmt.Sprintf("basic mytoken%d", index+1),
		SkipSSLValidation: true,
	}
}

func withThisManyBackends(numBackends int) {
	Context(fmt.Sprintf("With %d backends in the config", numBackends), func() {
		JustBeforeEach(func() {
			for i := 0; i < numBackends; i++ {
				toInsert := backendAtIndex(i)
				Expect(Commit(toInsert)).To(Succeed(), "error inserting %+v: %s", toInsert, err)
			}
		})

		Describe("Retrieving them with Get()", func() {
			var getBackend string       //The argument passed to Get()
			var retBackend *api.Backend //The result of the call to Get()

			AfterEach(func() {
				getBackend = ""
				retBackend = nil
			})

			JustBeforeEach(func() {
				Expect(getBackend).NotTo(BeEmpty(), "Did you forget to set the backend for a test?")
				retBackend = Get(getBackend)
			})

			testGetAtIndex := func(checkIndex int) {
				Context(fmt.Sprintf("When getting backend (index %d) that's in the config", checkIndex), func() {
					BeforeEach(func() {
						getBackend = backendAtIndex(checkIndex).Name
					})

					It("should return exactly what was inserted", func() {
						Expect(retBackend).NotTo(BeNil())
						Expect(*retBackend).To(Equal(*backendAtIndex(checkIndex)))
					})
				})
			}

			for checkIndex := 0; checkIndex < numBackends; checkIndex++ {
				testGetAtIndex(checkIndex)
			}

			Context("When retrieving a non-existent backend", func() {
				BeforeEach(func() {
					getBackend = "is_this_your_card"
				})

				It("should return nil", func() {
					Expect(retBackend).To(BeNil())
				})
			})
		})

		Describe("Retrieving them with List()", func() {
			var listedBackends []*api.Backend
			const maxBackends = 10

			JustBeforeEach(func() {
				listedBackends = List()
				sort.Slice(listedBackends, func(i, j int) bool {
					return listedBackends[i].Name < listedBackends[j].Name
				})
			})

			AfterEach(func() {
				listedBackends = nil
			})

			It("should return the correct number of backends", func() {
				Expect(len(listedBackends)).To(Equal(numBackends))
			})

			checkThisListedBackend := func(checkIndex int) {
				Describe(fmt.Sprintf("The backend at index %d in the return from List", checkIndex), func() {
					It("should match the backend that was inserted", func() {
						Expect(*listedBackends[checkIndex]).To(Equal(*backendAtIndex(checkIndex)))
					})
				})
			}

			for checkIndex := 0; checkIndex < len(listedBackends); checkIndex++ {
				checkThisListedBackend(checkIndex)
			}
		})

		Describe("Use()ing them", func() {
			testUsingThisIndex := func(useIndex int) {
				Context(fmt.Sprintf("Using the backend inserted at index %d", useIndex), func() {
					JustBeforeEach(func() {
						err = Use(backendAtIndex(useIndex).Name)
					})

					It("should not err", func() {
						Expect(err).NotTo(HaveOccurred())
					})

					Context("Retrieving with Current()", func() {
						var curBackend *api.Backend
						JustBeforeEach(func() {
							curBackend = Current()
						})

						It("should retrieve the currently used backend", func() {
							Expect(*curBackend).To(Equal(*backendAtIndex(useIndex)))
						})
					})

					Context("Delete()ing the current backend", func() {
						JustBeforeEach(func() {
							err = Delete(backendAtIndex(useIndex).Name)
						})

						It("should not err", func() {
							Expect(err).NotTo(HaveOccurred())
						})

						Describe("retrieving the current backend post-Delete()", func() {
							var curBackend *api.Backend
							JustBeforeEach(func() {
								curBackend = Current()
							})

							It("should return nil", func() {
								Expect(curBackend).To(BeNil())
							})
						})

						Describe("getting the backend through Get", func() {
							var gotBackend *api.Backend
							JustBeforeEach(func() {
								gotBackend = Get(backendAtIndex(useIndex).Name)
							})

							It("should return nil", func() {
								Expect(gotBackend).To(BeNil())
							})
						})
					})

					Context("Using the currently used backend", func() {
						JustBeforeEach(func() {
							err = Use(backendAtIndex(useIndex).Name)
						})

						It("should not err", func() {
							Expect(err).NotTo(HaveOccurred())
						})

						Specify("the Current() backend should be what's expected", func() {
							Expect(*Current()).To(Equal(*backendAtIndex(useIndex)))
						})
					})
				})
			}

			for useIndex := 0; useIndex < numBackends; useIndex++ {
				testUsingThisIndex(useIndex)
			}

			Context("Using a non-existent backend", func() {
				JustBeforeEach(func() {
					err = Use("is_this_your_card?")
				})

				It("should err", func() {
					Expect(err).To(HaveOccurred())
				})

				Specify("The current backend should be nil", func() {
					Expect(Current()).To(BeNil())
				})
			})
		})

		Describe("Delete()ing the inserted backends", func() {
			testDeletionAtIndex := func(deleteIndex int) {
				Context(fmt.Sprintf("Deleting backend at index %d", deleteIndex), func() {
					JustBeforeEach(func() {
						err = Delete(backendAtIndex(deleteIndex).Name)
					})

					It("should not have erred", func() {
						Expect(err).NotTo(HaveOccurred())
					})

					It("should not be gettable", func() {
						Expect(Get(backendAtIndex(deleteIndex).Name)).To(BeNil())
					})
				})
			}

			for deleteIndex := 0; deleteIndex < numBackends; deleteIndex++ {
				testDeletionAtIndex(deleteIndex)
			}

			Context("Delete()ing a non-existent backend", func() {
				JustBeforeEach(func() {
					err = Delete("how_about_this_one?")
				})

				It("should err", func() {
					Expect(err).To(HaveOccurred())
				})

				Specify("no backends should have been deleted", func() {
					Expect(len(List())).To(Equal(numBackends))
				})
			})
		})

		if numBackends > 0 {
			Context("After a deletion", func() {
				JustBeforeEach(func() {
					if numBackends == 0 {
						Skip("Can't delete with nothing inserted")
					}
					Delete(backendAtIndex(0).Name)
				})
				Specify(fmt.Sprintf("There should be the correct number of backends remaining in the config"), func() {
					Expect(len(List())).To(Equal(numBackends - 1))
				})
			})
		}
	})
}

var _ = Describe("Committing backends", func() {

	BeforeEach(func() { Initialize() })

	Context("When the backends are unique", func() {
		var numBackends int
		for _, numBackends = range [5]int{0, 1, 2, 10 /*, 100*/} {
			withThisManyBackends(numBackends)
		}
	})

	Context("When the backends have the same name", func() {
		JustBeforeEach(func() {
			Expect(Commit(backendAtIndex(0))).To(Succeed())
			second := backendAtIndex(1)
			second.Name = backendAtIndex(0).Name
			err = Commit(second)
		})

		It("should not err", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		Specify("The second one inserted should overwrite the first", func() {
			expected := backendAtIndex(1)
			expected.Name = backendAtIndex(0).Name
			Expect(*Get(backendAtIndex(0).Name)).To(Equal(*expected))
		})

		Specify("There should only be one version in the config", func() {
			Expect(len(List())).To(Equal(1))
		})
	})

	Context("When committing backends that are exactly the same", func() {
		JustBeforeEach(func() {
			Expect(Commit(backendAtIndex(0))).To(Succeed())
			err = Commit(backendAtIndex(0))
		})

		It("should not err", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should only commit one copy of the backend", func() {
			Expect(List()).To(HaveLen(1))
		})
	})

	Context("When the backend has an improper address", func() {
		var testAddress string
		JustBeforeEach(func() {
			toCommit := backendAtIndex(0)
			toCommit.Address = testAddress
			err = Commit(toCommit)
		})
		By("the address having no protocol scheme", func() {
			BeforeEach(func() {
				testAddress = "first.com"
			})

			It("should err", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		By("the address having an unsupported protocol scheme", func() {
			BeforeEach(func() {
				testAddress = "ssh://first.com"
			})

			It("should err", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
