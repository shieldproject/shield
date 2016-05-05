package supervisor_test

import (
	"github.com/google/go-github/github"
	"github.com/markbates/goth"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	. "github.com/starkandwayne/shield/supervisor"
)

var _ = Describe("GithubVerifier", func() {
	var gv *GithubVerifier

	BeforeEach(func() {
		gv = &GithubVerifier{Orgs: []string{"no-such-group"}}
	})
	Context("When Retrieving a user's membership", func() {
		var client *http.Client
		var proxy *FakeProxy
		var fakeSvr *ghttp.Server
		BeforeEach(func() {
			fakeSvr = ghttp.NewServer()
			proxy = &FakeProxy{Backend: fakeSvr, ResponseCode: http.StatusOK}
			client = &http.Client{Transport: proxy}
		})
		AfterEach(func() {
			proxy.Backend.Close()
		})
		It("Throws an error if github returned an error", func() {
			proxy.Backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/user/orgs", "page=1"),
					ghttp.RespondWith(http.StatusInternalServerError, ""),
				),
			)
			membership, err := gv.Membership(goth.User{}, client)
			Expect(err).Should(HaveOccurred())
			Expect(membership).Should(BeNil())
		})
		It("Returns user's membership if successful", func() {
			proxy.Backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/user/orgs", "page=1"),
					ghttp.RespondWithJSONEncoded(
						proxy.ResponseCode,
						[]github.Organization{
							{Login: github.String("test-org-1")},
							{Login: github.String("test-org-2")},
						},
						http.Header{
							"Link": []string{
								`<https://github.example.com/user/orgs?page=2>; rel="next"`,
								`<https://github.example.com/user/orgs?page=2>; rel="last"`,
							},
						},
					),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/user/orgs", "page=2"),
					ghttp.RespondWithJSONEncoded(
						proxy.ResponseCode,
						[]github.Organization{
							{Login: github.String("test-org-3")},
						},
						http.Header{
							"Link": []string{
								`<https://github.example.com/user/orgs?page=1>; rel="first"`,
								`<https://github.example.com/user/orgs?page=1>; rel="prev"`,
							},
						},
					),
				),
			)
			membership, err := gv.Membership(goth.User{}, client)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(membership).Should(Equal(map[string]interface{}{
				"Orgs": []string{"test-org-1", "test-org-2", "test-org-3"},
			}))
		})
	})
	Context("When Verifying a user's access", func() {
		It("Denies access if there are no orgs specified to allow access", func() {
			membership := map[string]interface{}{
				"Orgs": []string{"test-org-3"},
			}
			gv.Orgs = []string{}
			Expect(gv.Verify("user", membership)).To(BeFalse())
		})
		It("Denies access if 'Orgs' is not a string/interface slice", func() {
			gv.Orgs = []string{"test-org-3"}
			membership := map[string]interface{}{"Orgs": "test-org-3"}
			Expect(gv.Verify("user", membership)).To(BeFalse())
		})
		It("Denies access if 'Orgs' is an interface slice, but contains non-string values", func() {
			gv.Orgs = []string{"test-org-3"}
			membership := map[string]interface{}{
				"Orgs": []interface{}{1, 2, 3, "test-org-3"},
			}
			Expect(gv.Verify("user", membership)).To(BeFalse())
		})
		Context("And orgs is an interface slice", func() {
			It("Denies access if the user was not in an org in the list of allowed orgs", func() {
				membership := map[string]interface{}{
					"Orgs": []interface{}{"no-one-allowed"},
				}
				Expect(gv.Verify("user", membership)).To(BeFalse())
			})
			It("Grants access if the user was in an org in the list of allowed orgs", func() {
				gv.Orgs = []string{"test-org-3"}
				membership := map[string]interface{}{
					"Orgs": []interface{}{"test-org-3"},
				}
				Expect(gv.Verify("user", membership)).To(BeTrue())
			})
		})
		Context("And orgs is a string slice", func() {
			It("Denies access if the user was not in an org in the list of allowed orgs", func() {
				membership := map[string]interface{}{
					"Orgs": []string{"no-one-allowed"},
				}
				Expect(gv.Verify("user", membership)).To(BeFalse())
			})
			It("Grants access if the user was in an org in the list of allowed orgs", func() {
				gv.Orgs = []string{"test-org-3"}
				membership := map[string]interface{}{
					"Orgs": []string{"test-org-3"},
				}
				Expect(gv.Verify("user", membership)).To(BeTrue())
			})
		})
	})
})
var _ = Describe("UAAVerifier", func() {
	var uv *UAAVerifier
	BeforeEach(func() {
		uv = &UAAVerifier{Groups: []string{"no-such-group"}}
	})
	Context("When Verifying a user's access", func() {
		It("Denies access if there are no orgs specified to allow access", func() {
			membership := map[string]interface{}{
				"Groups": []string{"test-org"},
			}
			uv.Groups = []string{}
			Expect(uv.Verify("user", membership)).To(BeFalse())
		})
		It("Denies access if 'Groups' is not a string/interface slice", func() {
			uv.Groups = []string{"test-org"}
			membership := map[string]interface{}{"Groups": "test-org"}
			Expect(uv.Verify("user", membership)).To(BeFalse())
		})
		It("Denies access if 'Groups' is an interface slice, but contains non-string values", func() {
			uv.Groups = []string{"test-org"}
			membership := map[string]interface{}{
				"Groups": []interface{}{1, 2, 3, "test-org"},
			}
			Expect(uv.Verify("user", membership)).To(BeFalse())
		})
		Context("And orgs is an interface slice", func() {
			It("Denies access if the user was not in an org in the list of allowed orgs", func() {
				membership := map[string]interface{}{
					"Groups": []interface{}{"no-one-allowed"},
				}
				Expect(uv.Verify("user", membership)).To(BeFalse())
			})
			It("Grants access if the user was in an org in the list of allowed orgs", func() {
				uv.Groups = []string{"test-org"}
				membership := map[string]interface{}{
					"Groups": []interface{}{"test-org"},
				}
				Expect(uv.Verify("user", membership)).To(BeTrue())
			})
		})
		Context("And orgs is a string slice", func() {
			It("Denies access if the user was not in an org in the list of allowed orgs", func() {
				membership := map[string]interface{}{
					"Groups": []string{"no-one-allowed"},
				}
				Expect(uv.Verify("user", membership)).To(BeFalse())
			})
			It("Grants access if the user was in an org in the list of allowed orgs", func() {
				uv.Groups = []string{"test-org"}
				membership := map[string]interface{}{
					"Groups": []string{"test-org"},
				}
				Expect(uv.Verify("user", membership)).To(BeTrue())
			})
		})
	})
	Context("When retrieving a user's Membership", func() {
		var client *http.Client
		var proxy *FakeProxy
		var fakeSvr *ghttp.Server
		BeforeEach(func() {
			fakeSvr = ghttp.NewServer()
			proxy = &FakeProxy{Backend: fakeSvr, ResponseCode: http.StatusOK}
			client = &http.Client{Transport: proxy}
		})
		AfterEach(func() {
			proxy.Backend.Close()
		})
		It("Returns an error if the request for group info to UAA could not be created", func() {
			proxy.Backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/Groups", "attributes=displayName,members&filter=displayName=%22no-such-group%22"),
					ghttp.RespondWithJSONEncoded(proxy.ResponseCode, map[string]interface{}{
						"resources": []interface{}{
							map[string]interface{}{
								"displayName": "test-org",
								"members": []interface{}{
									map[string]interface{}{
										"value": "user-uuid",
									},
								},
							},
						},
					})))
			uv.UAA = "%"
			membership, err := uv.Membership(goth.User{}, client)
			Expect(err).Should(HaveOccurred())
			Expect(membership).Should(BeNil())
		})
		It("Returns an error if the request to the UAA failed", func() {
			proxy.Backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/Groups", "attributes=displayName,members&filter=displayName=%22no-such-group%22"),
					ghttp.RespondWithJSONEncoded(http.StatusInternalServerError, map[string]interface{}{
						"resources": []interface{}{
							map[string]interface{}{
								"displayName": "test-org",
								"members": []interface{}{
									map[string]interface{}{
										"value": "user-uuid",
									},
								},
							},
						},
					})))
			membership, err := uv.Membership(goth.User{}, client)
			Expect(err).Should(HaveOccurred())
			Expect(membership).Should(BeNil())
		})
		It("Returns an error if the response from the UAA does not contain 'resources' as an array", func() {
			proxy.Backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/Groups", "attributes=displayName,members&filter=displayName=%22no-such-group%22"),
					ghttp.RespondWithJSONEncoded(proxy.ResponseCode, map[string]interface{}{"resources": 123})))
			membership, err := uv.Membership(goth.User{}, client)
			Expect(err).Should(HaveOccurred())
			Expect(membership).Should(BeNil())
		})
		It("Returns an error if one of response's groups is not a map[string]interface{}", func() {
			proxy.Backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/Groups", "attributes=displayName,members&filter=displayName=%22no-such-group%22"),
					ghttp.RespondWithJSONEncoded(proxy.ResponseCode, map[string]interface{}{"resources": []interface{}{1, 2, 3}})))
			membership, err := uv.Membership(goth.User{}, client)
			Expect(err).Should(HaveOccurred())
			Expect(membership).Should(BeNil())
		})
		It("Returns an error if a group's DisplayName is not a string", func() {
			proxy.Backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/Groups", "attributes=displayName,members&filter=displayName=%22no-such-group%22"),
					ghttp.RespondWithJSONEncoded(proxy.ResponseCode, map[string]interface{}{
						"resources": []interface{}{
							map[string]interface{}{
								"displayName": 1234,
								"members": []interface{}{
									map[string]interface{}{
										"value": "user-uuid",
									},
								},
							},
						},
					})))
			membership, err := uv.Membership(goth.User{}, client)
			Expect(err).Should(HaveOccurred())
			Expect(membership).Should(BeNil())
		})
		It("Returns an error if group members is not an interface slice", func() {
			proxy.Backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/Groups", "attributes=displayName,members&filter=displayName=%22no-such-group%22"),
					ghttp.RespondWithJSONEncoded(proxy.ResponseCode, map[string]interface{}{
						"resources": []interface{}{
							map[string]interface{}{
								"displayName": "test-org",
								"members":     1234,
							},
						},
					})))
			membership, err := uv.Membership(goth.User{}, client)
			Expect(err).Should(HaveOccurred())
			Expect(membership).Should(BeNil())
		})
		It("Returns an error if any of the members are not map[string]interface{}", func() {
			proxy.Backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/Groups", "attributes=displayName,members&filter=displayName=%22no-such-group%22"),
					ghttp.RespondWithJSONEncoded(proxy.ResponseCode, map[string]interface{}{
						"resources": []interface{}{
							map[string]interface{}{
								"displayName": "test-org",
								"members":     []interface{}{1234},
							},
						},
					})))
			membership, err := uv.Membership(goth.User{}, client)
			Expect(err).Should(HaveOccurred())
			Expect(membership).Should(BeNil())
		})
		It("Returns an error if the member Value is not a string", func() {
			proxy.Backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/Groups", "attributes=displayName,members&filter=displayName=%22no-such-group%22"),
					ghttp.RespondWithJSONEncoded(proxy.ResponseCode, map[string]interface{}{
						"resources": []interface{}{
							map[string]interface{}{
								"displayName": "test-org",
								"members": []interface{}{
									map[string]interface{}{
										"value": 1234,
									},
								},
							},
						},
					})))

			membership, err := uv.Membership(goth.User{}, client)
			Expect(err).Should(HaveOccurred())
			Expect(membership).Should(BeNil())
		})
		It("Returns a map of groups->membership on success", func() {
			proxy.Backend.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/Groups", "attributes=displayName,members&filter=displayName=%22no-such-group%22"),
					ghttp.RespondWithJSONEncoded(proxy.ResponseCode, map[string]interface{}{
						"resources": []interface{}{
							map[string]interface{}{
								"displayName": "test-org",
								"members": []interface{}{
									map[string]interface{}{
										"value": "user-uuid",
									},
								},
							},
							map[string]interface{}{
								"displayName": "shouldNotMatch",
								"members": []interface{}{
									map[string]interface{}{
										"value": "not-the-right-user",
									},
								},
							},
						},
					})))
			membership, err := uv.Membership(goth.User{UserID: "user-uuid"}, client)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(membership).Should(Equal(map[string]interface{}{"Groups": []string{"test-org"}}))
		})

	})
})
