// The `s3` plugin for SHIELD is intended to be a back-end storage
// plugin, wrapping Amazon's Simple Storage Service (S3). It should
// be compatible with other services which emulate the S3 API, offering
// similar storage solutions for private cloud offerings (such as OpenStack
// Swift). However, this plugin has only been tested with Amazon S3.
//
// PLUGIN FEATURES
//
// This plugin implements functionality suitable for use with the following
// SHIELD Job components:
//
//  Target: no
//  Store:  yes
//
// PLUGIN CONFIGURATION
//
// The endpoint configuration passed to this plugin is used to determine
// how to connect to S3, and where to place/retrieve the data once connected.
// your endpoint JSON should look something like this:
//
//    {
//        "s3_host":             "s3.amazonaws.com", # default
//        "access_key_id":       "your-access-key-id",
//        "secret_access_key":   "your-secret-access-key",
//        "skip_ssl_validation":  false,
//        "bucket":              "bucket-name",
//        "prefix":              "/path/inside/bucket/to/place/backup/data",
//        "signature_version":   "4",  # should be 2 or 4. Defaults to 4
//        "socks5_proxy":        ""    # optionally defined SOCKS5 proxy to use for the s3 communications
//        "s3_port":             ""    # optionally defined port to use for the s3 communications
//    }
//
// Default Configuration
//
//    {
//        "s3_host"             : "s3.amazonawd.com",
//        "signature_version"   : "4",
//        "skip_ssl_validation" : false
//    }
//
// `prefix` will default to the empty string, and backups will be placed in the
// root of the bucket.
//
// The `s3_port` field is optional. If specified, `s3_host` cannot be empty.
//
// STORE DETAILS
//
// When storing data, this plugin connects to the S3 service, and uploads the data
// into the specified bucket, using a path/filename with the following format:
//
//    <prefix>/<YYYY>/<MM>/<DD>/<HH-mm-SS>-<UUID>
//
// Upon successful storage, the plugin then returns this filename to SHIELD to use
// as the `store_key` when the data needs to be retrieved, or purged.
//
// RETRIEVE DETAILS
//
// When retrieving data, this plugin connects to the S3 service, and retrieves the data
// located in the specified bucket, identified by the `store_key` provided by SHIELD.
//
// PURGE DETAILS
//
// When purging data, this plugin connects to the S3 service, and deletes the data
// located in the specified bucket, identified by the `store_key` provided by SHIELD.
//
// DEPENDENCIES
//
// None.
//
package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	minio "github.com/minio/minio-go"
	"github.com/starkandwayne/goutils/ansi"
	"golang.org/x/net/proxy"

	"github.com/starkandwayne/shield/plugin"
)

const (
	DefaultS3Host            = "s3.amazonaws.com"
	DefaultPrefix            = ""
	DefaultSigVersion        = "4"
	DefaultSkipSSLValidation = false
)

func validSigVersion(v string) bool {
	return v == "2" || v == "4"
}

func main() {
	p := S3Plugin{
		Name:    "S3 Backup + Storage Plugin",
		Author:  "Stark & Wayne",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
			Target: "no",
			Store:  "yes",
		},
		Example: `
{
  "access_key_id"       : "your-access-key-id",       # REQUIRED
  "secret_access_key"   : "your-secret-access-key",   # REQUIRED
  "bucket"              : "name-of-your-bucket",      # REQUIRED

  "s3_host"             : "s3.amazonaws.com",    # override Amazon S3 endpoint
  "s3_port"             : ""                     # optional port to access s3_host on
  "skip_ssl_validation" : false,                 # Skip certificate verification (not recommended)
  "prefix"              : "/path/in/bucket",     # where to store archives, inside the bucket
  "signature_version"   : "4",                   # AWS signature version; must be '2' or '4'
  "socks5_proxy"        : ""                     # optional SOCKS5 proxy for accessing S3
}
`,
		Defaults: `
{
  "s3_host"             : "s3.amazonawd.com",
  "signature_version"   : "4",
  "skip_ssl_validation" : false
}
`,
	}

	plugin.Run(p)
}

type S3Plugin plugin.PluginInfo

type S3ConnectionInfo struct {
	Host              string
	SkipSSLValidation bool
	AccessKey         string
	SecretKey         string
	Bucket            string
	PathPrefix        string
	SignatureVersion  string
	SOCKS5Proxy       string
	Port              string
}

func (p S3Plugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p S3Plugin) Validate(endpoint plugin.ShieldEndpoint) error {
	var (
		s    string
		err  error
		fail bool
	)

	s, err = endpoint.StringValueDefault("s3_host", DefaultS3Host)
	if err != nil {
		ansi.Printf("@R{\u2717 s3_host              %s}\n", err)
		fail = true
	} else {
		ansi.Printf("@G{\u2713 s3_host}              @C{%s}\n", s)
	}

	s, err = endpoint.StringValue("access_key_id")
	if err != nil {
		ansi.Printf("@R{\u2717 access_key_id        %s}\n", err)
		fail = true
	} else {
		ansi.Printf("@G{\u2713 access_key_id}        @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("s3_port", "")
	if err != nil {
		ansi.Printf("@R{\u2717 s3_port        %s}\n", err)
		fail = true
	} else {
		if s3Host, err := endpoint.StringValueDefault("s3_host", ""); s != "" && err == nil && s3Host == "" {
			ansi.Printf("@R{\u2717 s3_port        %s but s3_host cannot be empty}\n", s)
			fail = true
		} else {
			ansi.Printf("@G{\u2713 s3_port}        @C{%s}\n", s)
		}
	}

	s, err = endpoint.StringValue("secret_access_key")
	if err != nil {
		ansi.Printf("@R{\u2717 secret_access_key    %s}\n", err)
		fail = true
	} else {
		ansi.Printf("@G{\u2713 secret_access_key}    @C{%s}\n", s)
	}

	s, err = endpoint.StringValue("bucket")
	if err != nil {
		ansi.Printf("@R{\u2717 bucket               %s}\n", err)
		fail = true
	} else {
		ansi.Printf("@G{\u2713 bucket}               @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("prefix", DefaultPrefix)
	if err != nil {
		ansi.Printf("@R{\u2717 prefix               %s}\n", err)
		fail = true
	} else if s == "" {
		ansi.Printf("@G{\u2713 prefix}               (none)\n")
	} else {
		s = strings.TrimLeft(s, "/")
		ansi.Printf("@G{\u2713 prefix}               @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("signature_version", DefaultSigVersion)
	if err != nil {
		ansi.Printf("@R{\u2717 signature_version    %s}\n", err)
		fail = true
	} else if !validSigVersion(s) {
		ansi.Printf("@R{\u2717 signature_version    Unexpected signature version '%s' found (expecting '2' or '4')}\n", s)
		fail = true
	} else {
		ansi.Printf("@G{\u2713 signature_version}    @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("socks5_proxy", "")
	if err != nil {
		ansi.Printf("@R{\u2717 socks5_proxy         %s}\n", err)
		fail = true
	} else if s == "" {
		ansi.Printf("@G{\u2713 socks5_proxy}         (no proxy will be used)\n")
	} else {
		ansi.Printf("@G{\u2713 socks5_proxy}         @C{%s}\n", s)
	}

	tf, err := endpoint.BooleanValueDefault("skip_ssl_validation", DefaultSkipSSLValidation)
	if err != nil {
		ansi.Printf("@R{\u2717 skip_ssl_validation  %s}\n", err)
		fail = true
	} else if tf {
		ansi.Printf("@G{\u2713 skip_ssl_validation}  @C{yes}, SSL will @Y{NOT} be validated\n")
	} else {
		ansi.Printf("@G{\u2713 skip_ssl_validation}  @C{no}, SSL @Y{WILL} be validated\n")
	}

	if fail {
		return fmt.Errorf("s3: invalid configuration")
	}
	return nil
}

func (p S3Plugin) Backup(endpoint plugin.ShieldEndpoint) error {
	return plugin.UNIMPLEMENTED
}

func (p S3Plugin) Restore(endpoint plugin.ShieldEndpoint) error {
	return plugin.UNIMPLEMENTED
}

func (p S3Plugin) Store(endpoint plugin.ShieldEndpoint) (string, error) {
	s3, err := getS3ConnInfo(endpoint)
	if err != nil {
		return "", err
	}
	client, err := s3.Connect()
	if err != nil {
		return "", err
	}

	path := s3.genBackupPath()
	plugin.DEBUG("Storing data in %s", path)

	// FIXME: should we do something with the size of the write performed?
	// Removing leading slash until https://github.com/minio/minio/issues/3256 is fixed
	_, err = client.PutObject(s3.Bucket, strings.TrimPrefix(path, "/"), os.Stdin, "application/x-gzip")
	if err != nil {
		return "", err
	}

	return path, nil
}

func (p S3Plugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	s3, err := getS3ConnInfo(endpoint)
	if err != nil {
		return err
	}
	client, err := s3.Connect()
	if err != nil {
		return err
	}

	reader, err := client.GetObject(s3.Bucket, file)
	if err != nil {
		return err
	}
	if _, err = io.Copy(os.Stdout, reader); err != nil {
		return err
	}

	err = reader.Close()
	if err != nil {
		return err
	}

	return nil
}

func (p S3Plugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	s3, err := getS3ConnInfo(endpoint)
	if err != nil {
		return err
	}
	client, err := s3.Connect()
	if err != nil {
		return err
	}

	err = client.RemoveObject(s3.Bucket, file)
	if err != nil {
		return err
	}

	return nil
}

func getS3ConnInfo(e plugin.ShieldEndpoint) (S3ConnectionInfo, error) {
	host, err := e.StringValueDefault("s3_host", DefaultS3Host)
	if err != nil {
		return S3ConnectionInfo{}, err
	}

	insecure_ssl, err := e.BooleanValueDefault("skip_ssl_validation", DefaultSkipSSLValidation)
	if err != nil {
		return S3ConnectionInfo{}, err
	}

	key, err := e.StringValue("access_key_id")
	if err != nil {
		return S3ConnectionInfo{}, err
	}

	secret, err := e.StringValue("secret_access_key")
	if err != nil {
		return S3ConnectionInfo{}, err
	}

	bucket, err := e.StringValue("bucket")
	if err != nil {
		return S3ConnectionInfo{}, err
	}

	prefix, err := e.StringValueDefault("prefix", DefaultPrefix)
	if err != nil {
		return S3ConnectionInfo{}, err
	}
	prefix = strings.TrimLeft(prefix, "/")

	sigVer, err := e.StringValueDefault("signature_version", DefaultSigVersion)
	if !validSigVersion(sigVer) {
		return S3ConnectionInfo{}, fmt.Errorf("Invalid `signature_version` specified (`%s`). Expected `2` or `4`", sigVer)
	}

	proxy, err := e.StringValueDefault("socks5_proxy", "")
	if err != nil {
		return S3ConnectionInfo{}, err
	}

	port, err := e.StringValueDefault("s3_port", "")
	if err != nil {
		return S3ConnectionInfo{}, err
	}

	return S3ConnectionInfo{
		Host:              host,
		SkipSSLValidation: insecure_ssl,
		AccessKey:         key,
		SecretKey:         secret,
		Bucket:            bucket,
		PathPrefix:        prefix,
		SignatureVersion:  sigVer,
		SOCKS5Proxy:       proxy,
		Port:              port,
	}, nil
}

func (s3 S3ConnectionInfo) genBackupPath() string {
	t := time.Now()
	year, mon, day := t.Date()
	hour, min, sec := t.Clock()
	uuid := plugin.GenUUID()
	path := fmt.Sprintf("%s/%04d/%02d/%02d/%04d-%02d-%02d-%02d%02d%02d-%s", s3.PathPrefix, year, mon, day, year, mon, day, hour, min, sec, uuid)
	// Remove double slashes
	path = strings.Replace(path, "//", "/", -1)
	return path
}

func (s3 S3ConnectionInfo) Connect() (*minio.Client, error) {
	var s3Client *minio.Client
	var err error

	s3Host := s3.Host
	if s3.Port != "" {
		s3Host = s3.Host + ":" + s3.Port
	}

	if s3.SignatureVersion == "2" {
		s3Client, err = minio.NewV2(s3Host, s3.AccessKey, s3.SecretKey, false)
	} else {
		s3Client, err = minio.NewV4(s3Host, s3.AccessKey, s3.SecretKey, false)
	}
	if err != nil {
		return nil, err
	}

	transport := http.DefaultTransport
	transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: s3.SkipSSLValidation}
	if s3.SOCKS5Proxy != "" {
		dialer, err := proxy.SOCKS5("tcp", s3.SOCKS5Proxy, nil, proxy.Direct)
		if err != nil {
			fmt.Fprintln(os.Stderr, "can't connect to the proxy:", err)
			os.Exit(1)
		}
		transport.(*http.Transport).Dial = dialer.Dial
	}

	s3Client.SetCustomTransport(transport)

	return s3Client, nil
}
