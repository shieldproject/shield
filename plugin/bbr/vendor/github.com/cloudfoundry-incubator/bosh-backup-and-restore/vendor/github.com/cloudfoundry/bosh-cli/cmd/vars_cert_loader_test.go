package cmd_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	boshtpl "github.com/cloudfoundry/bosh-cli/director/template"
)

var _ = Describe("VarsCertLoader", func() {
	var (
		vars   boshtpl.StaticVariables
		loader VarsCertLoader
	)

	BeforeEach(func() {
		vars = boshtpl.StaticVariables{}
		loader = NewVarsCertLoader(vars)
	})

	Describe("LoadCerts", func() {
		It("returns error if getting variable failed", func() {
			loader = NewVarsCertLoader(&FakeVariables{GetErr: errors.New("fake-err")})

			_, _, err := loader.LoadCerts("unknown")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("returns error if variable by that name is not found", func() {
			_, _, err := loader.LoadCerts("unknown")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected to find variable 'unknown' with a certificate"))
		})

		It("returns error if variable cannot be parsed", func() {
			vars["cert"] = 123

			_, _, err := loader.LoadCerts("cert")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Expected variable 'cert' to be deserializable"))
		})

		const cert = "-----BEGIN CERTIFICATE-----\nMIIDtzCCAp+gAwIBAgIJAMZ/qRdRamluMA0GCSqGSIb3DQEBBQUAMEUxCzAJBgNV\nBAYTAkFVMRMwEQYDVQQIEwpTb21lLVN0YXRlMSEwHwYDVQQKExhJbnRlcm5ldCBX\naWRnaXRzIFB0eSBMdGQwIBcNMTYwODI2MjIzMzE5WhgPMjI5MDA2MTAyMjMzMTla\nMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIEwpTb21lLVN0YXRlMSEwHwYDVQQKExhJ\nbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAw\nggEKAoIBAQDN/bv70wDn6APMqiJZV7ESZhUyGu8OzuaeEfb+64SNvQIIME0s9+i7\nD9gKAZjtoC2Tr9bJBqsKdVhREd/X6ePTaopxL8shC9GxXmTqJ1+vKT6UxN4kHr3U\n+Y+LK2SGYUAvE44nv7sBbiLxDl580P00ouYTf6RJgW6gOuKpIGcvsTGA4+u0UTc+\ny4pj6sT0+e3xj//Y4wbLdeJ6cfcNTU63jiHpKc9Rgo4Tcy97WeEryXWz93rtRh8d\npvQKHVDU/26EkNsPSsn9AHNgaa+iOA2glZ2EzZ8xoaMPrHgQhcxoi8maFzfM2dX2\nXB1BOswa/46yqfzc4xAwaW0MLZLg3NffAgMBAAGjgacwgaQwHQYDVR0OBBYEFNRJ\nPYFebixALIR2Ee+yFoSqurxqMHUGA1UdIwRuMGyAFNRJPYFebixALIR2Ee+yFoSq\nurxqoUmkRzBFMQswCQYDVQQGEwJBVTETMBEGA1UECBMKU29tZS1TdGF0ZTEhMB8G\nA1UEChMYSW50ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkggkAxn+pF1FqaW4wDAYDVR0T\nBAUwAwEB/zANBgkqhkiG9w0BAQUFAAOCAQEAoPTwU2rm0ca5b8xMni3vpjYmB9NW\noSpGcWENbvu/p7NpiPAe143c5EPCuEHue/AbHWWxBzNAZvhVZBeFirYNB3HYnCla\njP4WI3o2Q0MpGy3kMYigEYG76WeZAM5ovl0qDP6fKuikZofeiygb8lPs7Hv4/88x\npSsZYBm7UPTS3Pl044oZfRJdqTpyHVPDqwiYD5KQcI0yHUE9v5KC0CnqOrU/83PE\nb0lpHA8bE9gQTQjmIa8MIpaP3UNTxvmKfEQnk5UAZ5xY2at5mmyj3t8woGdzoL98\nyDd2GtrGsguQXM2op+4LqEdHef57g7vwolZejJqN776Xu/lZtCTp01+HTA==\n-----END CERTIFICATE-----\n"
		const privKey = "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEAzf27+9MA5+gDzKoiWVexEmYVMhrvDs7mnhH2/uuEjb0CCDBN\nLPfouw/YCgGY7aAtk6/WyQarCnVYURHf1+nj02qKcS/LIQvRsV5k6idfryk+lMTe\nJB691PmPiytkhmFALxOOJ7+7AW4i8Q5efND9NKLmE3+kSYFuoDriqSBnL7ExgOPr\ntFE3PsuKY+rE9Pnt8Y//2OMGy3XienH3DU1Ot44h6SnPUYKOE3Mve1nhK8l1s/d6\n7UYfHab0Ch1Q1P9uhJDbD0rJ/QBzYGmvojgNoJWdhM2fMaGjD6x4EIXMaIvJmhc3\nzNnV9lwdQTrMGv+Osqn83OMQMGltDC2S4NzX3wIDAQABAoIBAArGTuLpMo7uz+QQ\nsiNCNvzjYhBw4DhCEkYKYoULBK/1RvnurNrBTOcb+Qzs8HbdfgTPmciCFMhDQw9a\ng/7jOQuB8yPggBuGZr2EVnr4/ERJQADAG26APSW6uAtrhaKRy62qtDDYEowMmr9J\nJSAaPmRWcPpsHsfJgWPYMKrwCvWvkwVIuIyGJdc993j/Dadh9c/YFdc/i6w8e2Xz\nFnVehTTJqtvZQM+C0AyfUPmneJ+ARSGK+vMtpZCGHhSwXgfoaTFAF3DvV7qfOBur\nqTja1BdYZDsxEiSIDExt90oyHO6lb2nA67SQoNJj9A6TWjioJriYAMTR2/nwjcu4\nM+1RkWECgYEA5kjGGmvICINxvRmNo0eL1peQonUBLpnosCnwlsNamVgpI0rfa+9w\nqWJyPjIY5+x9wIjNs9OYV6iQf/3A9rANk0jjDmZB01TeeQ5Pi65ZTDAX3YL6cKo2\n7PpvQU/nCFG1i/xxwdkRActewKg4ozaIRYMNVRIwOSf/J7i9Nb8W6GsCgYEA5P57\nxrw8iNclUuTpCBrKAbWP0VeIAu0iSIlf47CiKVOHrA8ycRElV71MZgMFqF8xEyD/\nnzW2r2XgfWXK+Qp5sLD6hJfM3zXNiTCpwaLqCUSLOVEvryf2ctYKc/oq7dpRwWkM\nHDn1O+VUs+7IvyVosfrVm8gc0yohZ1vz009de10CgYAfhp74RwEfiT8s8C6fx8+x\nFRbL5tC+nHtqgpNZUG06yQL4vetQT3tQ9RVGxnz6Yznj/daLY9BbT8xYeVjNbNSu\n8S+EbSNd1ySN1hO1v6yh7YOW47N9cRAL6U0J1/J9BRKhk3HPY/QcFsdmAKGgVnrZ\naVON7euEJ6GawoPEs+Bi+QKBgQDXnlvUBHiHbPWi+RIHZJojQ99Yga/6+WhXnqqg\njTgT66gLNgAMANYFqKPgRiY0pPVjiqXHNt9+hlH8ITYei2OMIQiygvEQl+uhqyWc\nw5bVBSqG3NAmgF2JQctz6vIzJmfm0s/pYBVuwYChMEzr1wCe3Y328lVZ7Aip9yY+\nKTPfrQKBgQCtFmolSFOJTyM/dwTt68MTM4/HlSC4cQOGBe37ug38omBIdInCEwD7\n8zLH2eSS5BqcpACmQ7QHkPL9ILDKmQB2Bwfl3fK58aHARrJ5jWRMXLITBv6KinaR\nhdU1xOQ3M9uKGDkggz4nlkOZgSXdszwcomTwn9j5XI6YpkG63xPbQQ==\n-----END RSA PRIVATE KEY----\n"

		It("returns error if pem encoded certificate cannot be found", func() {
			vars["cert"] = map[interface{}]interface{}{
				"certificate": "not-cert",
			}

			_, _, err := loader.LoadCerts("cert")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Certificate did not contain PEM formatted block"))
		})

		It("returns error if certificate cannot be parsed", func() {
			vars["cert"] = map[interface{}]interface{}{
				"certificate": "-----BEGIN CERTIFICATE-----\nMIIDtzCCAp+gAwIBAgIJAMZ/qRdR\n-----END CERTIFICATE-----\n",
			}

			_, _, err := loader.LoadCerts("cert")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Parsing certificate: asn1: syntax error: data truncated"))
		})

		It("returns error if pem encoded private key cannot be found", func() {
			vars["cert"] = map[interface{}]interface{}{
				"certificate": cert,
				"private_key": "not-priv-key",
			}

			_, _, err := loader.LoadCerts("cert")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Private key did not contain PEM formatted block"))
		})

		It("returns error if private key cannot be parsed", func() {
			vars["cert"] = map[interface{}]interface{}{
				"certificate": cert,
				"private_key": "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEAzf27+9MA5+gDzKoiWVex\n-----END RSA PRIVATE KEY-----\n",
			}

			_, _, err := loader.LoadCerts("cert")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Parsing private key: asn1: syntax error: data truncated"))
		})
	})
})

type FakeVariables struct {
	GetErr error
}

func (v *FakeVariables) Get(varDef boshtpl.VariableDefinition) (interface{}, bool, error) {
	return nil, false, v.GetErr
}

func (v *FakeVariables) List() ([]boshtpl.VariableDefinition, error) {
	return nil, nil
}
