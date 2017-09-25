package httpsdispatcher_test

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshdispatcher "github.com/cloudfoundry/bosh-agent/httpsdispatcher"
	fakelogger "github.com/cloudfoundry/bosh-agent/logger/fakes"
)

const targetURL = "https://user:pass@127.0.0.1:7789"

// Confirm the targetURL is valid and can be listened on before running tests.
func init() {
	u, err := url.Parse(targetURL)
	if err != nil {
		panic(fmt.Sprintf("Invalid target URL: %s", err))
	}
	ln, err := net.Listen("tcp", u.Host)
	if err != nil {
		panic(fmt.Sprintf("Unable to listen on address (%s): %s", targetURL, err))
	}
	ln.Close()
}

var _ = Describe("HTTPSDispatcher", func() {
	var (
		dispatcher *boshdispatcher.HTTPSDispatcher
		logger     *fakelogger.FakeLogger
	)

	BeforeEach(func() {
		logger = &fakelogger.FakeLogger{}
		serverURL, err := url.Parse(targetURL)
		Expect(err).ToNot(HaveOccurred())
		dispatcher = boshdispatcher.NewHTTPSDispatcher(serverURL, logger)

		errChan := make(chan error)
		go func() {
			errChan <- dispatcher.Start()
		}()

		select {
		case err := <-errChan:
			Expect(err).ToNot(HaveOccurred())
		case <-time.After(1 * time.Second):
			// server should now be running, continue
		}
	})

	AfterEach(func() {
		dispatcher.Stop()
		time.Sleep(1 * time.Second)
	})

	It("calls the handler function for the route", func() {
		var hasBeenCalled = false
		handler := func(w http.ResponseWriter, r *http.Request) {
			hasBeenCalled = true
			w.WriteHeader(201)
		}

		dispatcher.AddRoute("/example", handler)

		client := getHTTPClient()
		response, err := client.Get(targetURL + "/example")

		Expect(err).ToNot(HaveOccurred())
		Expect(response.StatusCode).To(BeNumerically("==", 201))
		Expect(hasBeenCalled).To(Equal(true))
	})

	It("returns a 404 if the route does not exist", func() {
		client := getHTTPClient()
		response, err := client.Get(targetURL + "/example")
		Expect(err).ToNot(HaveOccurred())
		Expect(response.StatusCode).To(BeNumerically("==", 404))
	})

	// Go's TLS client does not support SSLv3 (so we couldn't test it even if it did)
	PIt("does not allow connections using SSLv3", func() {
		handler := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }
		dispatcher.AddRoute("/example", handler)

		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionSSL30,
			MaxVersion:         tls.VersionSSL30,
		}
		client := getHTTPClientWithConfig(tlsConfig)
		_, err := client.Get(targetURL + "/example")
		Expect(err).To(HaveOccurred())
	})

	It("does allow connections using TLSv1", func() {
		handler := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }
		dispatcher.AddRoute("/example", handler)

		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS10,
			MaxVersion:         tls.VersionTLS10,
		}
		client := getHTTPClientWithConfig(tlsConfig)
		_, err := client.Get(targetURL + "/example")
		Expect(err).ToNot(HaveOccurred())
	})

	It("does allow connections using TLSv1.1", func() {
		handler := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }
		dispatcher.AddRoute("/example", handler)

		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS11,
			MaxVersion:         tls.VersionTLS11,
		}
		client := getHTTPClientWithConfig(tlsConfig)
		_, err := client.Get(targetURL + "/example")
		Expect(err).ToNot(HaveOccurred())
	})

	It("does allow connections using TLSv1.2", func() {
		handler := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }
		dispatcher.AddRoute("/example", handler)

		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
			MaxVersion:         tls.VersionTLS12,
		}
		client := getHTTPClientWithConfig(tlsConfig)
		_, err := client.Get(targetURL + "/example")
		Expect(err).ToNot(HaveOccurred())
	})

	It("does not allow connections using 3DES ciphers", func() {
		handler := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }
		dispatcher.AddRoute("/example", handler)

		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			CipherSuites: []uint16{
				tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
			},
		}
		client := getHTTPClientWithConfig(tlsConfig)
		_, err := client.Get(targetURL + "/example")
		Expect(err).To(HaveOccurred())
	})

	It("does not allow connections using RC4 ciphers", func() {
		handler := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }
		dispatcher.AddRoute("/example", handler)

		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			CipherSuites: []uint16{
				tls.TLS_RSA_WITH_RC4_128_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
				tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
			},
		}
		client := getHTTPClientWithConfig(tlsConfig)
		_, err := client.Get(targetURL + "/example")
		Expect(err).To(HaveOccurred())
	})

	It("does allow connections using AES ciphers", func() {
		handler := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }
		dispatcher.AddRoute("/example", handler)

		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			CipherSuites: []uint16{
				tls.TLS_RSA_WITH_AES_128_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			},
		}
		client := getHTTPClientWithConfig(tlsConfig)
		_, err := client.Get(targetURL + "/example")
		Expect(err).ToNot(HaveOccurred())
	})

	It("logs the request", func() {
		handler := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }
		dispatcher.AddRoute("/example", handler)
		client := getHTTPClient()
		_, err := client.Get(targetURL + "/example")
		Expect(err).ToNot(HaveOccurred())
		Expect(logger.InfoCallCount()).To(Equal(1))
		tag, message, _ := logger.InfoArgsForCall(0)
		Expect(message).To(Equal("GET /example"))
		Expect(tag).To(Equal("HTTPS Dispatcher"))
	})

	Context("When the basic authorization is wrong", func() {
		It("returns 401", func() {
			dispatcher.AddRoute("/example", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(500)
			})
			client := getHTTPClient()

			response, err := client.Get("https://bad:creds@127.0.0.1:7789/example")

			Expect(err).ToNot(HaveOccurred())
			Expect(response.StatusCode).To(BeNumerically("==", 401))
			Expect(response.Header.Get("WWW-Authenticate")).To(Equal(`Basic realm=""`))
		})
	})
})

func getHTTPClient() http.Client {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		// Both CBC & RC4 ciphers can be exploited
		// Mozilla's "Modern" recommended settings only overlap with the golang TLS client on these two ciphers
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
		// SSLv3 and TLSv1.0 are considered weak
		// TLS1.1 does not support GCM, so it won't actually be used
		MinVersion: tls.VersionTLS11,
		MaxVersion: tls.VersionTLS12,
	}
	return getHTTPClientWithConfig(tlsConfig)
}

func getHTTPClientWithConfig(tlsConfig *tls.Config) http.Client {
	httpTransport := &http.Transport{TLSClientConfig: tlsConfig}
	return http.Client{Transport: httpTransport}
}
