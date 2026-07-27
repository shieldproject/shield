// Package s3util provides shared utilities for S3-compatible storage plugins.
package s3util

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pborman/uuid"
	"golang.org/x/net/proxy"
)

// S3Config holds the connection parameters for an S3-compatible endpoint.
type S3Config struct {
	Region             string
	AccessKeyID        string
	SecretAccessKey    string
	SessionToken       string
	Endpoint           string // full URL, e.g. "https://s3.amazonaws.com"
	UsePathStyle       bool
	InsecureSkipVerify bool
	SOCKS5Proxy        string
}

// NewClient constructs an AWS SDK v2 S3 client from the provided S3Config.
func NewClient(cfg S3Config) (*s3.Client, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.InsecureSkipVerify, //nolint:gosec
		},
	}

	if cfg.SOCKS5Proxy != "" {
		dialer, err := proxy.SOCKS5("tcp", cfg.SOCKS5Proxy, nil, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("s3util: failed to create SOCKS5 dialer for %q: %w", cfg.SOCKS5Proxy, err)
		}
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		}
	}

	httpClient := &http.Client{Transport: transport}

	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}

	awsCfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				cfg.SessionToken,
			),
		),
		config.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, fmt.Errorf("s3util: failed to load AWS config: %w", err)
	}

	// Disable automatic checksum calculation for streaming uploads to
	// non-TLS endpoints (e.g. RustFS/MinIO over HTTP). The SDK default
	// (WhenSupported) requires a seekable body for trailing checksums,
	// which fails when piping stdin.
	awsCfg.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired

	var opts []func(*s3.Options)

	if cfg.Endpoint != "" {
		opts = append(opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	}

	if cfg.UsePathStyle {
		opts = append(opts, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}

	client := s3.NewFromConfig(awsCfg, opts...)
	return client, nil
}

// GenBackupPath generates a time-stamped, UUID-suffixed object key under base.
// The resulting path has the form:
//
//	base/YYYY/MM/DD/YYYY-MM-DD-HHmmSS-<uuid>
//
// Leading double-slashes are collapsed so an empty base produces a clean path.
func GenBackupPath(base string) string {
	t := time.Now()
	year, mon, day := t.Date()
	hour, min, sec := t.Clock()
	id := uuid.New()
	path := fmt.Sprintf(
		"%s/%04d/%02d/%02d/%04d-%02d-%02d-%02d%02d%02d-%s",
		base, year, mon, day, year, mon, day, hour, min, sec, id,
	)
	// Collapse any double-slashes that arise from an empty base.
	path = strings.ReplaceAll(path, "//", "/")
	return path
}

// CountingReader wraps an io.Reader and records how many bytes have been read.
type CountingReader struct {
	r io.Reader
	// N is the total number of bytes read so far.
	N int64
}

// NewCountingReader wraps r in a CountingReader.
func NewCountingReader(r io.Reader) *CountingReader {
	return &CountingReader{r: r}
}

// Read implements io.Reader and increments N by the number of bytes read.
func (c *CountingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.N += int64(n)
	return n, err
}
