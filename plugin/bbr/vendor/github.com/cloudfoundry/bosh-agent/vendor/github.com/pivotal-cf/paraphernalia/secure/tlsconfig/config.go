// Package tlsconfig provides opintionated helpers for building tls.Configs.
// It keeps up to date with internal Pivotal best practices and external
// industry best practices.
package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"
	"log"
)

// Config represents a half configured TLS configuration. It can be made usable
// by calling either of its two methods.
type Config struct {
	opts []TLSOption
}

// TLSOption can be used to configure a TLS configuration for both clients and
// servers.
type TLSOption func(*tls.Config)

// ServerOption can be used to configure a TLS configuration for a server.
type ServerOption func(*tls.Config)

// ClientOption can be used to configure a TLS configuration for a client.
type ClientOption func(*tls.Config)

// Build creates a half configured TLS configuration.
func Build(opts ...TLSOption) Config {
	return Config{
		opts: opts,
	}
}

// Server can be used to build a TLS configuration suitable for servers (GRPC,
// HTTP, etc.). The options are applied in order. It is possible for a later
// option to undo the configuration that an earlier one applied. Care must be
// taken.
func (c Config) Server(opts ...ServerOption) *tls.Config {
	config := &tls.Config{}

	for _, opt := range c.opts {
		opt(config)
	}

	for _, opt := range opts {
		opt(config)
	}

	return config
}

// Client can be used to build a TLS configuration suitable for clients (GRPC,
// HTTP, etc.). The options are applied in order. It is possible for a later
// option to undo the configuration that an earlier one applied. Care must be
// taken.
func (c Config) Client(opts ...ClientOption) *tls.Config {
	config := &tls.Config{}

	for _, opt := range c.opts {
		opt(config)
	}

	for _, opt := range opts {
		opt(config)
	}

	return config
}

// WithInternalServiceDefaults modifies a *tls.Config that is suitable for use
// in communication links between internal services. It is not guaranteed to be
// suitable for communication to other external services as it contains a
// strict definition of acceptable standards.
//
// The standards were taken from the "Consolidated Remarks" internal document
// from Pivotal.
//
// Note: Due to the aggressive nature of the ciphersuites chosen here (they do
// not support any ECC signing) it is not possible to use ECC keys with this
// option.
func WithInternalServiceDefaults() TLSOption {
	return func(c *tls.Config) {
		c.MinVersion = tls.VersionTLS12
		c.PreferServerCipherSuites = true
		c.CipherSuites = []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		}
		c.CurvePreferences = []tls.CurveID{
			tls.CurveP384,
		}
	}
}

// WithPivotalDefaults is the same as WithInternalServiceDefaults and is only
// provided for backwards compatibility. These configuration options are now
// used beyond just Pivotal.
//
// Deprecated: Use WithInternalServiceDefaults() instead.
func WithPivotalDefaults() TLSOption {
	log.Println("DEPRECATED! tlsconfig: Please use WithInternalServiceDefaults() rather than WithPivotalDefaults()")
	return WithInternalServiceDefaults()
}

// WithIdentity sets the identity of the server or client which will be
// presented to its peer upon connection.
func WithIdentity(cert tls.Certificate) TLSOption {
	return func(c *tls.Config) {
		c.Certificates = []tls.Certificate{cert}
	}
}

// WithClientAuthentication makes the server verify that all clients present an
// identity that can be validated by the certificate pool provided.
func WithClientAuthentication(authority *x509.CertPool) ServerOption {
	return func(c *tls.Config) {
		c.ClientAuth = tls.RequireAndVerifyClientCert
		c.ClientCAs = authority
	}
}

// WithAuthority makes the server verify that all clients present an identity
// that can be validated by the certificate pool provided.
func WithAuthority(authority *x509.CertPool) ClientOption {
	return func(c *tls.Config) {
		c.RootCAs = authority
	}
}
