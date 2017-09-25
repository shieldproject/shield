package types_test

import (
	. "github.com/cloudfoundry/config-server/types"

	"crypto/x509"
	"encoding/pem"
	"github.com/cloudfoundry/config-server/types/typesfakes"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func parseCertString(certString string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(certString))
	crt, err := x509.ParseCertificate(block.Bytes)

	return crt, err
}

func getCertResp(generator ValueGenerator, certParams map[interface{}]interface{}) CertResponse {
	certResp, err := generator.Generate(certParams)
	Expect(err).To(BeNil())

	return certResp.(CertResponse)
}

var _ = Describe("CertificateGenerator", func() {

	var (
		fakeLoader   *typesfakes.FakeCertsLoader
		generator    ValueGenerator
		fakeRootCert *x509.Certificate
	)

	mockCertValue := `-----BEGIN CERTIFICATE-----
MIICozCCAk2gAwIBAgIJAMCpfChXiHPFMA0GCSqGSIb3DQEBCwUAMGwxCzAJBgNV
BAYTAk5BMQ8wDQYDVQQIEwZOYXJuaWExFDASBgNVBAcTC1NwcmluZ2ZpZWxkMRYw
FAYDVQQKEw1GdXR1cmFtYSBDb3JwMR4wHAYDVQQDExVGdXR1cmFtYSBDb3JwIFJv
b3QgQ0EwHhcNMTcwMjAyMTUzNDQ3WhcNMzcwMTI4MTUzNDQ3WjBsMQswCQYDVQQG
EwJOQTEPMA0GA1UECBMGTmFybmlhMRQwEgYDVQQHEwtTcHJpbmdmaWVsZDEWMBQG
A1UEChMNRnV0dXJhbWEgQ29ycDEeMBwGA1UEAxMVRnV0dXJhbWEgQ29ycCBSb290
IENBMFwwDQYJKoZIhvcNAQEBBQADSwAwSAJBAJp1y8yYbM1VZC6okYpEjpJONwmM
yMBH0VhSsLwkFCZ79CoHBchr8TtUiRkSTzzuSunCJ0nQHR3ogfnU7SxIQqsCAwEA
AaOB0TCBzjAdBgNVHQ4EFgQUtnDDXajzWvC9+g2wpX47qmQnCEwwgZ4GA1UdIwSB
ljCBk4AUtnDDXajzWvC9+g2wpX47qmQnCEyhcKRuMGwxCzAJBgNVBAYTAk5BMQ8w
DQYDVQQIEwZOYXJuaWExFDASBgNVBAcTC1NwcmluZ2ZpZWxkMRYwFAYDVQQKEw1G
dXR1cmFtYSBDb3JwMR4wHAYDVQQDExVGdXR1cmFtYSBDb3JwIFJvb3QgQ0GCCQDA
qXwoV4hzxTAMBgNVHRMEBTADAQH/MA0GCSqGSIb3DQEBCwUAA0EAODPLqaoQOFj2
1Tm2m8kUrdT7yocHkZnfilYSE0w79QIT1okP/fpAKTHcl/rCa0g8t6qvNu7PHw26
bbok0ZsyJA==
-----END CERTIFICATE-----`

	mockKeyValue := `-----BEGIN RSA PRIVATE KEY-----
MIIBOwIBAAJBAJp1y8yYbM1VZC6okYpEjpJONwmMyMBH0VhSsLwkFCZ79CoHBchr
8TtUiRkSTzzuSunCJ0nQHR3ogfnU7SxIQqsCAwEAAQJAa659sxf8mjXSzvhz5nof
Dv56Pi0o82veFX4oejGI3r5sUZh5sKWaKMKKnu6OYaVI82Bm41+Pd0yPdf0m/ln3
YQIhAMn48p/QUsFbMblqRWu1PT6s+vwH3lXGNvgw+ao5rvhbAiEAw8c11JW3sEYs
+o+IuEAlwcvprU6Jps5fYw2KnFBzb/ECIQCChhi+ARJKFNY4nh4I8lKHG6EDmU4t
HnDNylC+mpKhuwIhAJ0Q20z7+GyBQGCcetFnFWOPFqAlnCWo97neCVAy8wnhAiAu
sHx2rlaLkmSreYJsmVaiSp0E9lhdympuDF+WKRolkQ==
-----END RSA PRIVATE KEY-----`

	BeforeEach(func() {
		fakeLoader = new(typesfakes.FakeCertsLoader)
		generator = NewCertificateGenerator(fakeLoader)

		cpb, _ := pem.Decode([]byte(mockCertValue))
		kpb, _ := pem.Decode([]byte(mockKeyValue))
		crt, _ := x509.ParseCertificate(cpb.Bytes)
		key, _ := x509.ParsePKCS1PrivateKey(kpb.Bytes)

		fakeLoader.LoadCertsReturns(crt, key, nil)

		fakeRootCert, _ = parseCertString(mockCertValue)
	})

	Describe("Generate", func() {
		var params map[interface{}]interface{}
		BeforeEach(func() {
			params = map[interface{}]interface{}{"common_name": "bosh.io"}
		})

		Context("when passed parameters types are NOT correct", func() {
			It("returns an error when CommonName is not of type string", func() {
				params["common_name"] = []int{1}
				_, err := generator.Generate(params)
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(Equal("Failed to generate certificate, parameters are invalid: Expected input to be deserializable: yaml: unmarshal errors:\n  line 2: cannot unmarshal !!seq into string"))
			})

			It("returns an error when AlternativeName is not of type []string", func() {
				params["alternative_names"] = "smurf"
				_, err := generator.Generate(params)
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(Equal("Failed to generate certificate, parameters are invalid: Expected input to be deserializable: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `smurf` into []string"))
			})

			It("returns an error when ca is not of type string", func() {
				params["ca"] = []int{1}
				_, err := generator.Generate(params)
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(Equal("Failed to generate certificate, parameters are invalid: Expected input to be deserializable: yaml: unmarshal errors:\n  line 2: cannot unmarshal !!seq into string"))
			})
		})

		Context("when passed parameters types are correct", func() {
			var params map[interface{}]interface{}
			BeforeEach(func() {
				params = map[interface{}]interface{}{}
			})

			Context("when 'is_ca' is TRUE", func() {
				BeforeEach(func() {
					params["is_ca"] = true
				})

				Context("when 'ca' is NOT set", func() {
					var certificate *x509.Certificate

					BeforeEach(func() {
						certResp := getCertResp(generator, params)
						certificate, _ = parseCertString(certResp.Certificate)
					})

					It("generates a root CA", func() {
						Expect(certificate.IsCA).To(BeTrue())
					})

					It("sets KeyUsage and ExtKeyUsage", func() {
						Expect(certificate.KeyUsage).To(Equal(x509.KeyUsageCertSign | x509.KeyUsageCRLSign))
						Expect(certificate.ExtKeyUsage).To(BeEmpty())
					})

					It("sets Issuer, Country & Org", func() {
						Expect(certificate.Issuer.Country).To(Equal([]string{"USA"}))
						Expect(certificate.Issuer.Organization).To(Equal([]string{"Cloud Foundry"}))
						Expect(certificate.Issuer.CommonName).To(Equal(""))
					})

				})

				Context("when 'ca' is NOT empty", func() {
					var certificate *x509.Certificate
					var certResp CertResponse
					BeforeEach(func() {
						params["ca"] = "smurf-cert"

						certResp = getCertResp(generator, params)
						certificate, _ = parseCertString(certResp.Certificate)
					})

					It("generates an intermediate CA cert", func() {
						Expect(certificate.IsCA).To(BeTrue())
					})

					It("sets KeyUsage and ExtKeyUsage", func() {
						Expect(certificate.KeyUsage).To(Equal(x509.KeyUsageCertSign | x509.KeyUsageCRLSign))
						Expect(certificate.ExtKeyUsage).To(BeEmpty())
					})

					It("sets Issuer Country & Org", func() {
						Expect(certificate.Issuer.Country).To(Equal([]string{"NA"}))
						Expect(certificate.Issuer.Organization).To(Equal([]string{"Futurama Corp"}))
						Expect(certificate.Issuer.CommonName).To(Equal("Futurama Corp Root CA"))
					})

					It("should be signed by the root CA", func() {
						certString := certResp.Certificate

						roots := x509.NewCertPool()
						success := roots.AppendCertsFromPEM([]byte(mockCertValue))
						Expect(success).To(BeTrue())

						block, _ := pem.Decode([]byte(certString))
						Expect(block).ToNot(BeNil())

						cert, err := x509.ParseCertificate(block.Bytes)
						Expect(err).To(BeNil())

						opts := x509.VerifyOptions{
							Roots: roots,
						}

						_, err = cert.Verify(opts)

						Expect(err).To(BeNil())
					})
				})
			})

			Context("when 'is_ca' is FALSE", func() {

				Context("when 'ca' is empty", func() {

					It("should throw an error", func() {
						_, err := generator.Generate(params)
						Expect(err).ToNot(BeNil())
						Expect(err.Error()).To(Equal("Missing required CA name"))
					})
				})

				Context("when 'ca' is NOT empty", func() {
					BeforeEach(func() {
						params["ca"] = "smurf-ca"
						params["common_name"] = "bosh.io"
					})
					It("generates a certificate", func() {
						certResp := getCertResp(generator, params)
						certificate, err := parseCertString(certResp.Certificate)

						Expect(err).To(BeNil())
						Expect(certificate).ToNot(BeNil())
					})

					It("sets KeyUsage", func() {
						altNames := []interface{}{"cloudfoundry.com", "example.com"}
						params["alternative_names"] = altNames
						certResp := getCertResp(generator, params)
						certificate, _ := parseCertString(certResp.Certificate)

						Expect(certificate.KeyUsage).To(Equal(x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature))
					})

					It("sets common name and alternative name as passed in", func() {
						altNames := []interface{}{"cloudfoundry.com", "example.com"}
						params["alternative_names"] = altNames
						certResp := getCertResp(generator, params)
						certificate, _ := parseCertString(certResp.Certificate)

						Expect(certificate.Subject.CommonName).Should(Equal("bosh.io"))

						Expect(certificate.DNSNames).ShouldNot(ContainElement("bosh.io"))
						Expect(certificate.DNSNames).Should(ContainElement("cloudfoundry.com"))
						Expect(certificate.DNSNames).Should(ContainElement("example.com"))
					})

					It("should work if CN was also included in SAN", func() {
						altNames := []interface{}{"bosh.io", "cloudfoundry.com", "example.com"}
						params["alternative_names"] = altNames
						certResp := getCertResp(generator, params)
						certificate, _ := parseCertString(certResp.Certificate)

						Expect(certificate.Subject.CommonName).Should(Equal("bosh.io"))

						Expect(certificate.DNSNames).Should(ContainElement("bosh.io"))
						Expect(certificate.DNSNames).Should(ContainElement("cloudfoundry.com"))
						Expect(certificate.DNSNames).Should(ContainElement("example.com"))
					})

					It("should set expiry for the cert in 1 year", func() {
						certResp := getCertResp(generator, params)
						certificate, _ := parseCertString(certResp.Certificate)

						oneYearFromToday := time.Now().UTC().Add(365 * 24 * time.Hour)

						Expect(certificate.NotAfter).Should(BeTemporally("~", oneYearFromToday, 5*time.Second))
					})

					It("should be signed by the parent CA", func() {
						certResp := getCertResp(generator, params)
						certString := certResp.Certificate

						roots := x509.NewCertPool()
						success := roots.AppendCertsFromPEM([]byte(mockCertValue))
						Expect(success).To(BeTrue())

						block, _ := pem.Decode([]byte(certString))
						Expect(block).ToNot(BeNil())

						cert, err := x509.ParseCertificate(block.Bytes)
						Expect(err).To(BeNil())

						opts := x509.VerifyOptions{
							Roots: roots,
						}

						_, err = cert.Verify(opts)

						Expect(err).To(BeNil())
					})

					It("is not a CA", func() {
						certResp := getCertResp(generator, params)
						certificate, _ := parseCertString(certResp.Certificate)

						Expect(certificate.IsCA).To(BeFalse())
					})

					It("generates a 3072-bit private key", func() {
						certResp := getCertResp(generator, params)

						Expect(certResp.PrivateKey).NotTo(BeEmpty())

						block, _ := pem.Decode([]byte(certResp.PrivateKey))
						key, _ := x509.ParsePKCS1PrivateKey(block.Bytes)

						Expect(key.PublicKey.N.BitLen()).To(Equal(3072))
					})

					It("should have the public keys of the private key and certificate match", func() {
						certResp := getCertResp(generator, params)
						certificate, _ := parseCertString(certResp.Certificate)

						block, _ := pem.Decode([]byte(certResp.PrivateKey))
						key, _ := x509.ParsePKCS1PrivateKey(block.Bytes)

						Expect(certificate.PublicKey).To(Equal(&key.PublicKey))
					})

					Context("when ExtKeyUsage is NOT empty", func() {
						Context("when it is client_auth", func() {
							It("should include the x509.ExtKeyUsageClientAuth flag in the key", func() {
								altNames := []interface{}{"cloudfoundry.com", "example.com"}
								params["alternative_names"] = altNames
								params["extended_key_usage"] = []string{"client_auth"}
								certResp := getCertResp(generator, params)
								certificate, _ := parseCertString(certResp.Certificate)

								Expect(certificate.ExtKeyUsage).To(Equal([]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}))
							})
						})

						Context("when it is server_auth", func() {
							It("should include the x509.ExtKeyUsageServerAuth flag in the key", func() {
								altNames := []interface{}{"cloudfoundry.com", "example.com"}
								params["alternative_names"] = altNames
								params["extended_key_usage"] = []string{"server_auth"}
								certResp := getCertResp(generator, params)
								certificate, _ := parseCertString(certResp.Certificate)

								Expect(certificate.ExtKeyUsage).To(Equal([]x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}))
							})
						})

						Context("when multiple auth types are set", func() {
							It("should include the x509.ExtKeyUsageServerAuth flag in the key", func() {
								altNames := []interface{}{"cloudfoundry.com", "example.com"}
								params["alternative_names"] = altNames
								params["extended_key_usage"] = []string{"client_auth", "server_auth"}
								certResp := getCertResp(generator, params)
								certificate, _ := parseCertString(certResp.Certificate)

								Expect(certificate.ExtKeyUsage).To(Equal([]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}))
							})
						})

						Context("when it is neither server or client auth", func() {
							It("returns an error", func() {
								altNames := []interface{}{"cloudfoundry.com", "example.com"}
								params["alternative_names"] = altNames
								params["extended_key_usage"] = []string{"something not supported"}
								_, err := generator.Generate(params)

								Expect(err).ToNot(BeNil())
								Expect(err.Error()).To(Equal("Unsupported extended key usage value: something not supported"))
							})
						})
					})

					Context("when ExtKeyUsage is empty", func() {
						It("should include the x509.ExtKeyUsageServerAuth flag in the key", func() {
							altNames := []interface{}{"cloudfoundry.com", "example.com"}
							params["alternative_names"] = altNames
							certResp := getCertResp(generator, params)
							certificate, _ := parseCertString(certResp.Certificate)

							Expect(certificate.ExtKeyUsage).To(Equal([]x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}))
						})
					})
				})
			})
		})

		Context("when passed parameters use unsupported keys", func() {
			var params map[interface{}]interface{}
			BeforeEach(func() {
				params = map[interface{}]interface{}{
					"is_ca":              true,
					"extended_key_usage": []string{"random", "values"},
					"ext_key_usage":      []string{"random", "values"},
				}
			})

			It("returns an error", func() {
				_, err := generator.Generate(params)

				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(Equal("Failed to generate certificate, parameters are invalid: Unsupported certificate parameter 'ext_key_usage'"))
			})
		})
	})
})
