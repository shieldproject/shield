package server_test

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"github.com/cloudfoundry/config-server/server"
	"github.com/cloudfoundry/config-server/store"
	. "github.com/cloudfoundry/config-server/store/storefakes"
	"github.com/cloudfoundry/config-server/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("x509Loader", func() {
	var (
		loader    types.CertsLoader
		mockStore *FakeStore
	)

	BeforeEach(func() {
		mockStore = &FakeStore{}
		loader = server.NewX509Loader(mockStore)
	})

	Context("when certificate is present in the store", func() {
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
			certResponse := types.CertResponse{
				Certificate: mockCertValue,
				PrivateKey:  mockKeyValue,
				CA:          "some-ca",
			}
			serializedCertResponse, _ := json.Marshal(certResponse)

			respValues := []store.Configuration{
				{
					Value: "{\"value\":" + string(serializedCertResponse) + "}",
				},
			}
			mockStore.GetByNameReturns(respValues, nil)
		})
		It("return a parsed certificate", func() {
			cpb, _ := pem.Decode([]byte(mockCertValue))
			kpb, _ := pem.Decode([]byte(mockKeyValue))
			expectedCrt, _ := x509.ParseCertificate(cpb.Bytes)
			expectedKey, _ := x509.ParsePKCS1PrivateKey(kpb.Bytes)

			actualCrt, actualKey, err := loader.LoadCerts("some-name")

			Expect(err).To(BeNil())
			Expect(mockStore.GetByNameArgsForCall(0)).To(Equal("some-name"))
			Expect(actualCrt).To(Equal(expectedCrt))
			Expect(actualKey).To(Equal(expectedKey))

		})
	})

	Context("when certificate is NOT present in the store", func() {

		BeforeEach(func() {
			mockStore.GetByNameReturns([]store.Configuration{}, nil)
		})
		It("it should throw an error", func() {
			_, _, err := loader.LoadCerts("some-name")

			Expect(err).ToNot(BeNil())
		})
	})

	Context("when the key is malformed JSON", func() {
		respValues := []store.Configuration{
			{
				Value: `{"value:"common value"}`,
			},
		}
		BeforeEach(func() {
			mockStore.GetByNameReturns(respValues, nil)
		})
		It("it should throw an error", func() {
			_, _, err := loader.LoadCerts("some-name")

			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("Failed to parse certificate"))
		})
	})

	Context("when the key is not of type certificate", func() {
		respValues := []store.Configuration{
			{
				Value: `{"value":"common value"}`,
			},
		}
		BeforeEach(func() {
			mockStore.GetByNameReturns(respValues, nil)
		})
		It("it should throw an error", func() {
			_, _, err := loader.LoadCerts("some-name")

			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("Failed to parse certificate"))
		})
	})

	Context("when the one of the certificate fields is empty", func() {
		respValues := []store.Configuration{
			{
				Value: `{"value":{"ca":"common value", "private_key": "some-private-key"}}`,
			},
		}
		BeforeEach(func() {
			mockStore.GetByNameReturns(respValues, nil)
		})
		It("it should throw an error", func() {
			_, _, err := loader.LoadCerts("some-name")

			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("Certificate some-name doesn't contain expected attributes"))
		})
	})

})
