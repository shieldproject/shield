package httpsdispatcher

import (
	"crypto/subtle"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

const httpsDispatcherLogTag = "HTTPS Dispatcher"

type HTTPSDispatcher struct {
	httpServer                  *http.Server
	mux                         *http.ServeMux
	listener                    net.Listener
	logger                      boshlog.Logger
	baseURL                     *url.URL
	expectedAuthorizationHeader string
}

type HTTPHandlerFunc func(writer http.ResponseWriter, request *http.Request)

func NewHTTPSDispatcher(baseURL *url.URL, logger boshlog.Logger) *HTTPSDispatcher {
	tlsConfig := &tls.Config{
		// SSLv3 is insecure due to BEAST and POODLE attacks
		MinVersion: tls.VersionTLS10,
		// Both 3DES & RC4 ciphers can be exploited
		// Using Mozilla's "Modern" recommended settings (where they overlap with golang support)
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
		PreferServerCipherSuites: true,
	}
	httpServer := &http.Server{
		TLSConfig: tlsConfig,
	}
	mux := http.NewServeMux()
	httpServer.Handler = mux

	expectedUsername := baseURL.User.Username()
	expectedPassword, _ := baseURL.User.Password()
	auth := fmt.Sprintf("%s:%s", expectedUsername, expectedPassword)
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
	expectedAuthorizationHeader := fmt.Sprintf("Basic %s", encodedAuth)

	return &HTTPSDispatcher{
		httpServer:                  httpServer,
		mux:                         mux,
		logger:                      logger,
		baseURL:                     baseURL,
		expectedAuthorizationHeader: expectedAuthorizationHeader,
	}
}

func (h *HTTPSDispatcher) Start() error {
	tcpListener, err := net.Listen("tcp", h.baseURL.Host)
	if err != nil {
		return bosherr.WrapError(err, "Starting HTTP listener")
	}
	h.listener = tcpListener

	cert, err := tls.LoadX509KeyPair("agent.cert", "agent.key")
	if err != nil {
		return bosherr.WrapError(err, "Loading agent SSL cert")
	}

	// update the server config with the cert
	config := h.httpServer.TLSConfig
	config.NextProtos = []string{"http/1.1"}
	config.Certificates = []tls.Certificate{cert}

	tlsListener := tls.NewListener(tcpListener, config)

	return h.httpServer.Serve(tlsListener)
}

func (h *HTTPSDispatcher) Stop() {
	if h.listener != nil {
		_ = h.listener.Close()
		h.listener = nil
	}
}

func (h *HTTPSDispatcher) requestNotAuthorized(request *http.Request) bool {
	return h.constantTimeEquals(h.expectedAuthorizationHeader, request.Header.Get("Authorization"))
}

func (h *HTTPSDispatcher) constantTimeEquals(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) != 1
}

func (h *HTTPSDispatcher) AddRoute(route string, handler HTTPHandlerFunc) {
	authWrapper := func(w http.ResponseWriter, r *http.Request) {
		h.logger.Info(httpsDispatcherLogTag, fmt.Sprintf("%s %s", r.Method, r.URL.Path))

		if h.requestNotAuthorized(r) {
			w.Header().Add("WWW-Authenticate", `Basic realm=""`)
			w.WriteHeader(401)
			return
		}

		handler(w, r)
	}

	h.mux.HandleFunc(route, authWrapper)
}
