package net_test

import (
	. "github.com/cloudfoundry/bosh-cli/common/net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ClientFactoryConfig", func() {
	Describe("Validate", func() {
		It("returns without error for basic config", func() {
			err := ClientFactoryConfig{Host: "host", Port: 1, Client: "client"}.Validate()
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error if host is empty", func() {
			err := ClientFactoryConfig{}.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Missing 'Host'"))
		})

		It("returns error if port is empty", func() {
			err := ClientFactoryConfig{Host: "host"}.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Missing 'Port'"))
		})

		It("returns error if client is empty", func() {
			err := ClientFactoryConfig{Host: "host", Port: 1}.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Missing 'Client'"))
		})

		It("returns error if cannot parse PEM formatted block", func() {
			err := ClientFactoryConfig{
				Host:   "host",
				Port:   1,
				Client: "client",
				CACert: "-",
			}.Validate()

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Parsing certificate 1: Missing PEM block"))
		})
	})

	Describe("CACertPool", func() {
		It("returns error if cannot parse PEM formatted block", func() {
			_, err := ClientFactoryConfig{
				Host:   "host",
				Port:   1,
				Client: "client",
				CACert: "-",
			}.CACertPool()

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Parsing certificate 1: Missing PEM block"))
		})

		It("does not create a cert pool from an empty string", func() {
			caCert := ``

			certPool, err := ClientFactoryConfig{CACert: caCert}.CACertPool()
			Expect(err).ToNot(HaveOccurred())
			Expect(certPool).To(BeNil())
		})

		It("parses the certificate", func() {
			caCert := `-----BEGIN CERTIFICATE-----
MIIDXzCCAkegAwIBAgIJAPerMgLAne5vMA0GCSqGSIb3DQEBBQUAMEUxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX
aWRnaXRzIFB0eSBMdGQwIBcNMTYwMTE2MDY0NTA0WhgPMjI4OTEwMzAwNjQ1MDRa
MEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJ
bnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAw
ggEKAoIBAQCtSo3KPjnVPzodb6+mNwbCdcpzVop8OmfwJ3ynQtyBEzGaKsAn4tlz
/wfQQrKFHgxqVpqcoxAlWPNMs5+iO2Jst3Gz2+oLcaDyz/EWorw0iF5q1F6+WYHp
EijY20MzaWYMyu4UhhlbJCkSGZSjujh5SFOAXQwWYJXsqjyxA9KaTD6OdH5Kpger
B9D4zogX0We00eouyvvz/sAeDbTshk9sJRGWHNFJr+TjVx2D01alU49liAL94yF6
1eEOEbE50OAhv9RNsRh6O58idaHg30bbMf1yAzcgBvh8CzIHH0BPofoF2pRfztoY
uudZ0ftJjTz4fA2h/7GOVzxemrTjx88vAgMBAAGjUDBOMB0GA1UdDgQWBBQjz5Q2
YW2kBTb4XLqKFZMSBLpi6zAfBgNVHSMEGDAWgBQjz5Q2YW2kBTb4XLqKFZMSBLpi
6zAMBgNVHRMEBTADAQH/MA0GCSqGSIb3DQEBBQUAA4IBAQA/s94M/mSGELHJWIb1
oE0IKHWajBd3Pc8+O1TZRE+ke3q+rZRfcxd2dAjq6zQHJUs2+fs0B3DyT9Wtyyoq
UrRdsgprOdf2Cuw8bMIsCQOvqWKhhdlLTnCi2xaGJawGsIkheuD1n+Il9gRQ2WGy
lACxVngPwjNYxjOE+CUnSZCuAmAfQYzqto3bNPqkgEwb7ueODeOiyhR8SKsH7ySW
QAOSxgrLBblGLWcDF9fjMeYaUnI34pHviCKeVxfgsxDR+Jg11F78sPdYLOF6ipBe
/5qTYucsY20B2EKtlscD0mSYBRwbVrSQt2RYbTCwaibxWUC13VV+YEk0NAv9Mm04
6sKO
-----END CERTIFICATE-----`

			certPool, err := ClientFactoryConfig{CACert: caCert}.CACertPool()
			Expect(err).ToNot(HaveOccurred())
			Expect(certPool.Subjects()[0]).To(ContainSubstring("Internet Widgits Pty Ltd"))
		})
	})
})
