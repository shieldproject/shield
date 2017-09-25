package server_test

import (
	"errors"
	. "github.com/cloudfoundry/config-server/server"
	. "github.com/cloudfoundry/config-server/server/serverfakes"
	. "github.com/cloudfoundry/config-server/store/storefakes"
	. "github.com/cloudfoundry/config-server/types/typesfakes"
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"encoding/json"
	"github.com/cloudfoundry/config-server/store"
	"github.com/cloudfoundry/config-server/types"
	"io"
)

type BadMockStore struct{}

func (store BadMockStore) Get(name string) (string, error) {
	return "", errors.New("")
}

func (store BadMockStore) Put(name string, value string) error {
	return errors.New("")
}

func generateHTTPRequest(method, urlStr string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return nil, err
	}

	methodsWithContentType := []string{"PUT", "POST"}

	if stringInSlice(method, methodsWithContentType) {
		req.Header.Add("Content-Type", "application/json")
	}

	return req, nil
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

var _ = Describe("RequestHandlerConcrete", func() {

	Describe("Given a nil store", func() {

		Context("creating the requestHandler", func() {
			It("should return an error", func() {
				_, err := NewRequestHandler(nil, types.NewValueGeneratorConcrete(&FakeCertsLoader{}))
				Expect(err.Error()).To(Equal("Data store must be set"))
			})
		})
	})

	Describe("Given a server with store", func() {
		var requestHandler http.Handler
		var mockTokenValidator *FakeTokenValidator
		var mockStore *FakeStore
		var mockValueGeneratorFactory *FakeValueGeneratorFactory
		var mockValueGenerator *FakeValueGenerator

		BeforeEach(func() {
			mockTokenValidator = &FakeTokenValidator{}
			mockStore = &FakeStore{}
			mockValueGeneratorFactory = &FakeValueGeneratorFactory{}
			mockValueGenerator = &FakeValueGenerator{}
			requestHandler, _ = NewRequestHandler(mockStore, mockValueGeneratorFactory)
		})

		Context("when URL path is invalid", func() {
			It("should return 400 Bad Request", func() {
				invalidPaths := []string{"/v1", "/v1/", "/v1/data", "/v1/data?name="}
				validMethods := []string{"GET", "PUT", "POST", "DELETE"}

				for _, method := range validMethods {
					for _, path := range invalidPaths {
						req, _ := generateHTTPRequest(method, path, nil)
						recorder := httptest.NewRecorder()
						requestHandler.ServeHTTP(recorder, req)

						Expect(recorder.Code).To(Equal(http.StatusBadRequest))
					}
				}
			})

			Context("when name query param is missing", func() {
				It("should return 400 Bad Request", func() {
					validMethods := []string{"GET", "DELETE"}
					for _, method := range validMethods {
						req, _ := generateHTTPRequest(method, "/v1/data?name=", nil)
						getRecorder := httptest.NewRecorder()
						requestHandler.ServeHTTP(getRecorder, req)

						Expect(getRecorder.Code).To(Equal(http.StatusBadRequest))
					}
				})
			})

			Context("when name format is invalid", func() {
				It("should return 400 Bad Request", func() {
					inValidURLPaths := []string{
						"/v1/data?name=name%2F%7B%2F*",   // /v1/data/name/{/*
						"/v1/data?name=name%2F%40%3F%2F", // /v1/data/name/@?/
					}

					validMethods := []string{"GET", "DELETE"}

					for _, method := range validMethods {
						for _, path := range inValidURLPaths {
							req, _ := generateHTTPRequest(method, path, nil)
							recorder := httptest.NewRecorder()
							requestHandler.ServeHTTP(recorder, req)

							Expect(recorder.Code).To(Equal(http.StatusBadRequest))
							Expect(recorder.Body.String()).To(ContainSubstring("Name must consist of alphanumeric, underscores, dashes, and forward slashes"))
						}
					}
				})
			})
		})

		Context("when URL path is valid", func() {

			Context("when http method is not supported", func() {
				It("should return 405 Method Not Allowed", func() {
					req, _ := generateHTTPRequest("PATCH", "/v1/data?name=bla", nil)
					recorder := httptest.NewRecorder()
					requestHandler.ServeHTTP(recorder, req)

					Expect(recorder.Code).To(Equal(http.StatusMethodNotAllowed))
				})
			})

			Context("when http method is supported", func() {
				Describe("/v1/data", func() {
					Describe("GET", func() {

						Context("when configuration with id exists", func() {
							It("returns value in the store", func() {
								respValue := store.Configuration{
									Value: `{"value":"crossfit"}`,
									Name:  "bla",
									ID:    "some_id",
								}
								mockStore.GetByIDReturns(respValue, nil)

								getReq, _ := generateHTTPRequest("GET", "/v1/data/"+respValue.ID, nil)
								getRecorder := httptest.NewRecorder()
								requestHandler.ServeHTTP(getRecorder, getReq)

								Expect(getRecorder.Code).To(Equal(http.StatusOK))
								expectedString := `{"id":"some_id","name":"bla","value":"crossfit"}`
								Expect(getRecorder.Body.String()).To(Equal(expectedString))
							})
						})

						Context("when configuration with id does not exist", func() {
							It("should return 404 Not Found", func() {
								req, _ := generateHTTPRequest("GET", "/v1/data/5", nil)
								getRecorder := httptest.NewRecorder()
								requestHandler.ServeHTTP(getRecorder, req)

								Expect(getRecorder.Code).To(Equal(http.StatusNotFound))
							})
						})
					})
				})

				Describe("/v1/data?name=<configuration name>", func() {
					validURLPaths := map[string]string{
						"/v1/data?name=smurf":                                "smurf",
						"/v1/data?name=smurf%2Fcolor":                        "smurf/color",
						"/v1/data?name=smurf%2Fcolor%2Fdarkness":             "smurf/color/darkness",
						"/v1/data?name=smurf%2Fcolor%2Fdark_ness%2Fname-tag": "smurf/color/dark_ness/name-tag",
					}
					Describe("GET", func() {

						It("can handle all types of valid names", func() {
							respValues := []store.Configuration{
								{
									Value: `{"value":"common value"}`,
								},
							}
							mockStore.GetByNameReturns(respValues, nil)
							var counter int = 0
							for path, extractedName := range validURLPaths {
								getReq, _ := generateHTTPRequest("GET", path, nil)
								getRecorder := httptest.NewRecorder()

								requestHandler.ServeHTTP(getRecorder, getReq)
								name := mockStore.GetByNameArgsForCall(counter)

								Expect(name).To(Equal(extractedName))

								Expect(getRecorder.Code).To(Equal(http.StatusOK))
								counter = counter + 1
							}
						})

						Context("when name exists", func() {
							It("returns value in the store", func() {
								values := []string{
									`123`,
									`"blabla"`,
									`{"name":"blabla"}`,
								}

								for _, value := range values {
									respValues := []store.Configuration{
										{
											Value: `{"value":` + value + "}",
											Name:  "bla",
											ID:    "some_id",
										},
									}

									mockStore.GetByNameReturns(respValues, nil)

									getReq, _ := generateHTTPRequest("GET", "/v1/data?name=bla", nil)
									getRecorder := httptest.NewRecorder()
									requestHandler.ServeHTTP(getRecorder, getReq)

									Expect(getRecorder.Code).To(Equal(http.StatusOK))
									expectedString := `{"data":[{"id":"some_id","name":"bla","value":` + value + "}]}"
									Expect(getRecorder.Body.String()).To(Equal(expectedString))
								}
							})
						})

						Context("when name does not exist", func() {
							It("should return 404 Not Found", func() {
								req, _ := generateHTTPRequest("GET", "/v1/data?name=test", nil)
								getRecorder := httptest.NewRecorder()
								requestHandler.ServeHTTP(getRecorder, req)

								Expect(getRecorder.Code).To(Equal(http.StatusNotFound))
							})
						})

						Context("when store errors", func() {
							It("returns 500 Internal Server Error", func() {
								mockStore.GetByNameReturns([]store.Configuration{}, errors.New("Kaboom!"))

								getReq, _ := generateHTTPRequest("GET", "/v1/data?name=bla", nil)
								getRecorder := httptest.NewRecorder()
								requestHandler.ServeHTTP(getRecorder, getReq)

								Expect(getRecorder.Code).To(Equal(http.StatusInternalServerError))
							})
						})
					})

					Describe("PUT", func() {
						It("throws an error if request header content type is not application/json", func() {
							req, _ := http.NewRequest("PUT", "/v1/data/", strings.NewReader(`{"value":"str"}`))
							putRecorder := httptest.NewRecorder()
							requestHandler.ServeHTTP(putRecorder, req)

							Expect(putRecorder.Body.String()).To(ContainSubstring("Unsupported Media Type - Accepts application/json only"))
							Expect(putRecorder.Code).To(Equal(http.StatusUnsupportedMediaType))
						})

						Context("when request body is NOT in the specified format", func() {
							Context("when body is empty", func() {
								It("should return 400 Bad Request", func() {
									req, _ := generateHTTPRequest("PUT", "/v1/data", nil)
									putRecorder := httptest.NewRecorder()
									requestHandler.ServeHTTP(putRecorder, req)

									Expect(putRecorder.Body.String()).To(ContainSubstring("Request can't be empty"))
									Expect(putRecorder.Code).To(Equal(http.StatusBadRequest))
								})
							})

							Context("when body is NOT JSON string", func() {
								It("should return 400 Bad Request", func() {
									req, _ := generateHTTPRequest("PUT", "/v1/data", strings.NewReader(`smurf`))
									putRecorder := httptest.NewRecorder()
									requestHandler.ServeHTTP(putRecorder, req)

									Expect(putRecorder.Body.String()).To(ContainSubstring("Request Body should be JSON string"))
									Expect(putRecorder.Code).To(Equal(http.StatusBadRequest))
								})
							})

							Context("when name is missing in the body", func() {
								It("should return 400 Bad Request", func() {
									req, _ := generateHTTPRequest("PUT", "/v1/data", strings.NewReader(`{"value":"blue"}`))
									putRecorder := httptest.NewRecorder()
									requestHandler.ServeHTTP(putRecorder, req)

									Expect(putRecorder.Body.String()).To(ContainSubstring("JSON request body should contain the key 'name'"))
									Expect(putRecorder.Code).To(Equal(http.StatusBadRequest))
								})
							})

							Context("when name is NOT of type string", func() {
								It("should return 400 Bad Request", func() {
									req, _ := generateHTTPRequest("PUT", "/v1/data", strings.NewReader(`{"name":{"foo":"bar"},"value":"james"}`))
									recorder := httptest.NewRecorder()
									requestHandler.ServeHTTP(recorder, req)

									Expect(recorder.Body.String()).To(ContainSubstring("JSON request body key 'name' must be of type string"))
									Expect(recorder.Code).To(Equal(http.StatusBadRequest))
								})
							})

							Context("when value is missing in the body", func() {
								It("should return 400 Bad Request", func() {
									req, _ := generateHTTPRequest("PUT", "/v1/data?name=some-name", strings.NewReader(`{"name":"smurf"}`))
									putRecorder := httptest.NewRecorder()
									requestHandler.ServeHTTP(putRecorder, req)

									Expect(putRecorder.Body.String()).To(ContainSubstring("JSON request body should contain the key 'value'"))
									Expect(putRecorder.Code).To(Equal(http.StatusBadRequest))
								})
							})

						})

						Context("when request body is in the specified format", func() {

							BeforeEach(func() {
								config := store.Configuration{
									Name:  "bla",
									Value: `{"value":"burpees"}`,
									ID:    "1",
								}
								mockStore.GetByIDReturns(config, nil)
								mockStore.PutReturns(config.ID, nil)
							})

							It("returns value, name and id in the response", func() {
								req, _ := generateHTTPRequest("PUT", "/v1/data", strings.NewReader(`{"name":"bla","value":"str"}`))
								putRecorder := httptest.NewRecorder()
								requestHandler.ServeHTTP(putRecorder, req)

								name := mockStore.GetByIDArgsForCall(0)
								Expect(name).To(Equal("1"))

								Expect(putRecorder.Body.String()).To(Equal(`{"id":"1","name":"bla","value":"burpees"}`))
							})

							Context("when value is a string ", func() {
								It("should store value in a specific JSON format and respond with 204 StatusNoContent", func() {
									req, _ := generateHTTPRequest("PUT", "/v1/data", strings.NewReader(`{"name":"bla","value":"str"}`))
									putRecorder := httptest.NewRecorder()
									requestHandler.ServeHTTP(putRecorder, req)

									Expect(mockStore.PutCallCount()).To(Equal(1))
									name, value := mockStore.PutArgsForCall(0)

									Expect(name).To(Equal("bla"))
									Expect(value).To(Equal(`{"value":"str"}`))
									Expect(putRecorder.Code).To(Equal(http.StatusOK))
								})
							})

							Context("when value is a number", func() {
								It("should store value in a specific JSON format and respond with 204 StatusNoContent", func() {
									req, _ := generateHTTPRequest("PUT", "/v1/data", strings.NewReader(`{"name":"bla","value":123}`))
									putRecorder := httptest.NewRecorder()
									requestHandler.ServeHTTP(putRecorder, req)

									Expect(mockStore.PutCallCount()).To(Equal(1))
									name, value := mockStore.PutArgsForCall(0)

									Expect(name).To(Equal("bla"))
									Expect(value).To(Equal(`{"value":123}`))
									Expect(putRecorder.Code).To(Equal(http.StatusOK))
								})
							})

							Context("when value is a JSON hash", func() {
								It("should store value in a specific JSON format and respond with 204 StatusNoContent", func() {
									requestBody := `{"name":"bla","value":{"age":10,"color":"red"}}`
									valueToStore := `{"value":{"age":10,"color":"red"}}`

									req, _ := generateHTTPRequest("PUT", "/v1/data", strings.NewReader(requestBody))
									putRecorder := httptest.NewRecorder()
									requestHandler.ServeHTTP(putRecorder, req)

									Expect(mockStore.PutCallCount()).To(Equal(1))
									name, value := mockStore.PutArgsForCall(0)

									Expect(name).To(Equal("bla"))
									Expect(value).To(Equal(valueToStore))
									Expect(putRecorder.Code).To(Equal(http.StatusOK))
								})
							})
						})
					})

					Describe("POST", func() {
						It("throws an error if request header content type is not application/json", func() {
							req, _ := http.NewRequest("POST", "/v1/data", strings.NewReader(`{"name":"somename","type":"password","parameters":{}}`))
							postRecorder := httptest.NewRecorder()
							requestHandler.ServeHTTP(postRecorder, req)

							Expect(postRecorder.Body.String()).To(ContainSubstring("Unsupported Media Type - Accepts application/json only"))
							Expect(postRecorder.Code).To(Equal(http.StatusUnsupportedMediaType))
						})

						Context("when request body is NOT in the specified format", func() {
							Context("when body is empty", func() {
								It("should return 400 Bad Request", func() {
									req, _ := generateHTTPRequest("POST", "/v1/data", nil)
									recorder := httptest.NewRecorder()
									requestHandler.ServeHTTP(recorder, req)

									Expect(recorder.Body.String()).To(ContainSubstring("Request can't be empty"))
									Expect(recorder.Code).To(Equal(http.StatusBadRequest))
								})
							})

							Context("when body is NOT JSON string", func() {
								It("should return 400 Bad Request", func() {
									req, _ := generateHTTPRequest("POST", "/v1/data", strings.NewReader("smurf"))
									recorder := httptest.NewRecorder()
									requestHandler.ServeHTTP(recorder, req)

									Expect(recorder.Body.String()).To(ContainSubstring("Request Body should be JSON string"))
									Expect(recorder.Code).To(Equal(http.StatusBadRequest))
								})
							})

							Context("when name is missing in the body", func() {
								It("should return 400 Bad Request", func() {
									req, _ := generateHTTPRequest("POST", "/v1/data", strings.NewReader(`{"value":"james"}`))
									recorder := httptest.NewRecorder()
									requestHandler.ServeHTTP(recorder, req)

									Expect(recorder.Body.String()).To(ContainSubstring("JSON request body should contain the key 'name'"))
									Expect(recorder.Code).To(Equal(http.StatusBadRequest))
								})
							})

							Context("when name is NOT of type string", func() {
								It("should return 400 Bad Request", func() {
									req, _ := generateHTTPRequest("POST", "/v1/data", strings.NewReader(`{"name":{"foo":"bar"},"type":"password"}`))
									recorder := httptest.NewRecorder()
									requestHandler.ServeHTTP(recorder, req)

									Expect(recorder.Body.String()).To(ContainSubstring("JSON request body key 'name' must be of type string"))
									Expect(recorder.Code).To(Equal(http.StatusBadRequest))
								})
							})

							Context("when type is missing in the body", func() {
								It("should return 400 Bad Request", func() {
									req, _ := generateHTTPRequest("POST", "/v1/data", strings.NewReader(`{"name":"bond"}`))
									recorder := httptest.NewRecorder()
									requestHandler.ServeHTTP(recorder, req)

									Expect(recorder.Body.String()).To(ContainSubstring("JSON request body should contain the key 'type'"))
									Expect(recorder.Code).To(Equal(http.StatusBadRequest))
								})
							})

							Context("when type is NOT of type string", func() {
								It("should return 400 Bad Request", func() {
									req, _ := generateHTTPRequest("POST", "/v1/data", strings.NewReader(`{"name":"moop","type":2}`))
									recorder := httptest.NewRecorder()
									requestHandler.ServeHTTP(recorder, req)

									Expect(recorder.Body.String()).To(ContainSubstring("JSON request body key 'type' must be of type string"))
									Expect(recorder.Code).To(Equal(http.StatusBadRequest))
								})
							})
						})

						Context("when request body is in the specified format", func() {

							Describe("Password generation", func() {
								Context("when value already exists", func() {
									It("should not generate a password", func() {
										mockStore.GetByNameStub = func(name string) (store.Configurations, error) {
											respValues := store.Configurations{
												{
													Value: `{"value":"smurf"}`,
													ID:    "some_id",
													Name:  "bla",
												},
											}
											return respValues, nil
										}

										postReq, _ := generateHTTPRequest("POST", "/v1/data", strings.NewReader(`{"name":"bla", "type":"password","parameters":{}}`))

										recorder := httptest.NewRecorder()
										requestHandler.ServeHTTP(recorder, postReq)

										Expect(recorder.Code).To(Equal(http.StatusOK))
										Expect(recorder.Body.String()).To(Equal(`{"id":"some_id","name":"bla","value":"smurf"}`))
										Expect(mockValueGeneratorFactory.GetGeneratorCallCount()).To(Equal(0))
									})
								})

								Context("when value does NOT exist", func() {
									It("should return generated password", func() {
										requestHandler, _ = NewRequestHandler(store.NewMemoryStore(), types.NewValueGeneratorConcrete(&FakeCertsLoader{}))

										postReq, _ := generateHTTPRequest("POST", "/v1/data", strings.NewReader(`{"name":"bla","type":"password","parameters":{}}`))

										recorder := httptest.NewRecorder()
										requestHandler.ServeHTTP(recorder, postReq)

										Expect(recorder.Code).To(Equal(http.StatusCreated))

										var data map[string]string
										json.Unmarshal(recorder.Body.Bytes(), &data)

										Expect(data["name"]).To(Equal("bla"))
										Expect(data["value"]).Should(MatchRegexp("[a-z0-9]{20}"))
									})
								})
							})

							Describe("Certificate generation", func() {
								Context("when value already exists", func() {
									It("should not generate certificates", func() {

										mockStore.GetByNameStub = func(name string) (store.Configurations, error) {
											respValue := store.Configurations{
												{
													Value: `{"value":"smurf"}`,
													ID:    "some_id",
													Name:  "bla",
												},
											}
											return respValue, nil
										}

										postReq, _ := generateHTTPRequest("POST", "/v1/data", strings.NewReader(`{"name":"bla","type":"certificate","parameters":{}}`))

										recorder := httptest.NewRecorder()
										requestHandler.ServeHTTP(recorder, postReq)

										Expect(recorder.Code).To(Equal(http.StatusOK))
										Expect(recorder.Body.String()).To(Equal(`{"id":"some_id","name":"bla","value":"smurf"}`))
										Expect(mockValueGeneratorFactory.GetGeneratorCallCount()).To(Equal(0))
									})
								})

								Context("when value does NOT exist", func() {
									It("should return generated certificate, its private key and root certificate used to sign the generated certificate", func() {
										requestHandler, _ = NewRequestHandler(store.NewMemoryStore(), mockValueGeneratorFactory)
										mockValueGeneratorFactory.GetGeneratorReturns(mockValueGenerator, nil)

										mockValueGenerator.GenerateReturns(types.CertResponse{
											Certificate: "fake-certificate",
											PrivateKey:  "fake-private-key",
											CA:          "fake-ca",
										}, nil)

										postReq, _ := generateHTTPRequest("POST", "/v1/data", strings.NewReader(`{"name":"bla","type":"certificate","parameters":{"common_name": "asdf", "alternative_names":["nam1", "name2"]}}`))

										recorder := httptest.NewRecorder()
										requestHandler.ServeHTTP(recorder, postReq)

										Expect(recorder.Code).To(Equal(http.StatusCreated))

										var data map[string]interface{}
										json.Unmarshal(recorder.Body.Bytes(), &data)

										Expect(data["name"]).To(Equal("bla"))

										value := data["value"].(map[string]interface{})
										Expect(value["certificate"]).To(Equal("fake-certificate"))
										Expect(value["private_key"]).To(Equal("fake-private-key"))
										Expect(value["ca"]).To(Equal("fake-ca"))
									})
								})
							})
						})
					})

					Describe("DELETE", func() {
						It("can handle all types of valid names", func() {
							mockStore.DeleteReturns(1, nil)

							var counter int = 0
							for name, extractedName := range validURLPaths {
								req, _ := generateHTTPRequest("DELETE", name, nil)

								recorder := httptest.NewRecorder()
								requestHandler.ServeHTTP(recorder, req)

								Expect(mockStore.DeleteArgsForCall(counter)).To(Equal(extractedName))
								counter++
							}
						})

						Context("Name exists", func() {
							BeforeEach(func() {
								mockStore.DeleteReturns(1, nil)
							})

							It("should delete all entries with given name", func() {
								req, _ := generateHTTPRequest("DELETE", "/v1/data?name=bla", nil)

								putRecorder := httptest.NewRecorder()
								requestHandler.ServeHTTP(putRecorder, req)

								Expect(mockStore.DeleteCallCount()).To(Equal(1))
								Expect(mockStore.DeleteArgsForCall(0)).To(Equal("bla"))
							})

							It("should return 204 Status No Content", func() {
								req, _ := generateHTTPRequest("DELETE", "/v1/data?name=bla", nil)
								req.Header.Set("Authorization", "bearer fake-auth-header")

								putRecorder := httptest.NewRecorder()
								requestHandler.ServeHTTP(putRecorder, req)

								Expect(putRecorder.Code).To(Equal(http.StatusNoContent))
							})
						})

						Context("Name does not exist", func() {
							It("should return 404 Status Not Found", func() {
								req, _ := generateHTTPRequest("DELETE", "/v1/data?name=bla", nil)
								req.Header.Set("Authorization", "bearer fake-auth-header")

								putRecorder := httptest.NewRecorder()
								requestHandler.ServeHTTP(putRecorder, req)

								Expect(putRecorder.Code).To(Equal(http.StatusNotFound))
								Expect(mockStore.DeleteCallCount()).To(Equal(1))
								Expect(mockStore.DeleteArgsForCall(0)).To(Equal("bla"))
							})
						})
					})
				})
			})
		})
	})
})
