package director_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/bosh-cli/common/net"
	. "github.com/cloudfoundry/bosh-cli/director"
)

var _ = Describe("NewConfigFromURL", func() {
	It("sets host and port (25555) if scheme is specified", func() {
		config, err := NewConfigFromURL("https://host")
		Expect(err).ToNot(HaveOccurred())
		genericConfig := net.ClientFactoryConfig{Host: "host", Port: 25555}
		Expect(config).To(Equal(Config{ClientFactoryConfig: genericConfig}))
	})

	It("sets host and port (25555) if scheme is not specified", func() {
		config, err := NewConfigFromURL("host")
		Expect(err).ToNot(HaveOccurred())
		genericConfig := net.ClientFactoryConfig{Host: "host", Port: 25555}
		Expect(config).To(Equal(Config{ClientFactoryConfig: genericConfig}))
	})

	It("extracts port if scheme is specified", func() {
		config, err := NewConfigFromURL("https://host:4443")
		Expect(err).ToNot(HaveOccurred())
		genericConfig := net.ClientFactoryConfig{Host: "host", Port: 4443}
		Expect(config).To(Equal(Config{ClientFactoryConfig: genericConfig}))
	})

	It("extracts port if scheme is not specified", func() {
		config, err := NewConfigFromURL("host:4443")
		Expect(err).ToNot(HaveOccurred())
		genericConfig := net.ClientFactoryConfig{Host: "host", Port: 4443}
		Expect(config).To(Equal(Config{ClientFactoryConfig: genericConfig}))
	})

	It("works with ipv6 hosts", func() {
		config, err := NewConfigFromURL("https://[2600:1f17:a63:5c00:5a20:7eec:cf9:e31f]:25555")
		Expect(err).ToNot(HaveOccurred())
		genericConfig := net.ClientFactoryConfig{Host: "2600:1f17:a63:5c00:5a20:7eec:cf9:e31f", Port: 25555}
		Expect(config).To(Equal(Config{ClientFactoryConfig: genericConfig}))
	})

	It("returns error if url is empty", func() {
		_, err := NewConfigFromURL("")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Expected non-empty Director URL"))
	})

	It("returns error if host is not specified", func() {
		_, err := NewConfigFromURL("https://:25555")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Expected to extract host from"))
	})

	It("returns error if parsing url fails", func() {
		_, err := NewConfigFromURL(":/")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Parsing Director URL"))
	})

	It("returns error if port cannot be extracted", func() {
		_, err := NewConfigFromURL("https://host::")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Extracting host/port from URL"))
	})

	It("returns error if port is empty", func() {
		_, err := NewConfigFromURL("host:")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Extracting port from URL"))
	})

	It("returns error if port cannot be parsed as int", func() {
		_, err := NewConfigFromURL("https://host:abc")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Extracting port from URL"))
	})
})
