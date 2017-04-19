// The `scality` plugin for SHIELD is intended to be a back-end storage
// plugin, wrapping the S3 API that Scality exposes, slightly altered to work
// within the confines of the ways that Scality differs from S3. It uses the
// version 2 signature version, and places a lower limit on the part size
// of a multipart object `put` operation.
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
// how to connect to Scality, and where to place/retrieve the data once connected.
// your endpoint JSON should look something like this:
//
//    {
//        "scality_host":        "your-scality-host",
//        "access_key_id":       "your-access-key-id",
//        "secret_access_key":   "your-secret-access-key",
//        "skip_ssl_validation":  false,
//        "bucket":              "bucket-name",
//        "prefix":              "/path/inside/bucket/to/place/backup/data",
//        "socks5_proxy":        "" #optionally defined SOCKS5 proxy to use for the scality communications
//    }
//
// Default Configuration
//
//    {
//        "skip_ssl_validation" : false,
//        "prefix"              : "",
//        "socks5_proxy"        : ""
//    }
//
// `prefix` will default to the empty string, and backups will be placed in the
// root of the bucket.
//
// STORE DETAILS
//
// When storing data, this plugin connects to the Scality service, and uploads the data
// into the specified bucket, using a path/filename with the following format:
//
//    <prefix>/<YYYY>/<MM>/<DD>/<HH-mm-SS>-<UUID>
//
// Upon successful storage, the plugin then returns this filename to SHIELD to use
// as the `store_key` when the data needs to be retrieved, or purged.
//
// RETRIEVE DETAILS
//
// When retrieving data, this plugin connects to the Scality service, and retrieves the data
// located in the specified bucket, identified by the `store_key` provided by SHIELD.
//
// PURGE DETAILS
//
// When purging data, this plugin connects to the Scality service, and deletes the data
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
	"time"

	"github.com/starkandwayne/goutils/ansi"
	minio "github.com/starkandwayne/minio-go"
	"golang.org/x/net/proxy"

	"github.com/starkandwayne/shield/plugin"
)

const (
	DefaultPrefix            = ""
	DefaultSkipSSLValidation = false
)

func main() {
	p := ScalityPlugin{
		Name:    "Scality Backup + Storage Plugin",
		Author:  "Stark & Wayne",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
			Target: "no",
			Store:  "yes",
		},
		Example: `
{
  "scality_host"        : "your-scality-host",       # REQUIRED
  "access_key_id"       : "your-access-key-id",      # REQUIRED
  "secret_access_key"   : "your-secret-access-key",  # REQUIRED
  "bucket"              : "name-of-your-bucket",     # REQUIRED

  "skip_ssl_validation" : false,                 # Skip certificate verification (not recommended)
  "prefix"              : "/path/in/bucket",     # where to store archives, inside the bucket
  "socks5_proxy"        : ""                     # optional SOCKS5 proxy for accessing S3
}
`,
		Defaults: `
{
  "skip_ssl_validation" : false,  # Always verify certificates
  "prefix"              : "",     # store archives in the root
  "socks5_proxy"        : ""      # don't use a proxy
}
`,
	}

	plugin.Run(p)
}

type ScalityPlugin plugin.PluginInfo

type ScalityConnectionInfo struct {
	Host              string
	SkipSSLValidation bool
	AccessKey         string
	SecretKey         string
	Bucket            string
	PathPrefix        string
	SOCKS5Proxy       string
}

func (p ScalityPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p ScalityPlugin) Validate(endpoint plugin.ShieldEndpoint) error {
	var (
		s    string
		err  error
		fail bool
	)

	s, err = endpoint.StringValue("scality_host")
	if err != nil {
		ansi.Printf("@R{\u2717 scality_host              %s}\n", err)
		fail = true
	} else {
		ansi.Printf("@G{\u2713 scality_host}              @C{%s}\n", s)
	}

	s, err = endpoint.StringValue("access_key_id")
	if err != nil {
		ansi.Printf("@R{\u2717 access_key_id        %s}\n", err)
		fail = true
	} else {
		ansi.Printf("@G{\u2713 access_key_id}        @C{%s}\n", s)
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
		ansi.Printf("@G{\u2713 prefix}               @C{%s}\n", s)
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
		return fmt.Errorf("scality: invalid configuration")
	}
	return nil
}

func (p ScalityPlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	return plugin.UNIMPLEMENTED
}

func (p ScalityPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	return plugin.UNIMPLEMENTED
}

func (p ScalityPlugin) Store(endpoint plugin.ShieldEndpoint) (string, error) {
	scal, err := getScalityConnInfo(endpoint)
	if err != nil {
		return "", err
	}
	client, err := scal.Connect()
	if err != nil {
		return "", err
	}

	path := scal.genBackupPath()
	plugin.DEBUG("Storing data in %s", path)

	minio.SetMaxPartSize(1024 * 1024 * 75)
	// FIXME: should we do something with the size of the write performed?
	_, err = client.PutObject(scal.Bucket, path, os.Stdin, "application/x-gzip")
	if err != nil {
		return "", err
	}

	return path, nil
}

func (p ScalityPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	scal, err := getScalityConnInfo(endpoint)
	if err != nil {
		return err
	}
	client, err := scal.Connect()
	if err != nil {
		return err
	}

	reader, err := client.GetObject(scal.Bucket, file)
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

func (p ScalityPlugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	scal, err := getScalityConnInfo(endpoint)
	if err != nil {
		return err
	}
	client, err := scal.Connect()
	if err != nil {
		return err
	}

	err = client.RemoveObject(scal.Bucket, file)
	if err != nil {
		return err
	}

	return nil
}

func getScalityConnInfo(e plugin.ShieldEndpoint) (ScalityConnectionInfo, error) {
	host, err := e.StringValue("scality_host")
	if err != nil {
		return ScalityConnectionInfo{}, err
	}

	insecure_ssl, err := e.BooleanValueDefault("skip_ssl_validation", DefaultSkipSSLValidation)
	if err != nil {
		return ScalityConnectionInfo{}, err
	}

	key, err := e.StringValue("access_key_id")
	if err != nil {
		return ScalityConnectionInfo{}, err
	}

	secret, err := e.StringValue("secret_access_key")
	if err != nil {
		return ScalityConnectionInfo{}, err
	}

	bucket, err := e.StringValue("bucket")
	if err != nil {
		return ScalityConnectionInfo{}, err
	}

	prefix, err := e.StringValueDefault("prefix", DefaultPrefix)
	if err != nil {
		return ScalityConnectionInfo{}, err
	}

	proxy, err := e.StringValueDefault("socks5_proxy", "")
	if err != nil {
		return ScalityConnectionInfo{}, err
	}

	return ScalityConnectionInfo{
		Host:              host,
		SkipSSLValidation: insecure_ssl,
		AccessKey:         key,
		SecretKey:         secret,
		Bucket:            bucket,
		PathPrefix:        prefix,
		SOCKS5Proxy:       proxy,
	}, nil
}

func (scal ScalityConnectionInfo) genBackupPath() string {
	t := time.Now()
	year, mon, day := t.Date()
	hour, min, sec := t.Clock()
	uuid := plugin.GenUUID()
	return fmt.Sprintf("%s/%04d/%02d/%02d/%04d-%02d-%02d-%02d%02d%02d-%s", scal.PathPrefix, year, mon, day, year, mon, day, hour, min, sec, uuid)
}

func (scal ScalityConnectionInfo) Connect() (*minio.Client, error) {
	// github.com/starkandwayne/minio-go has the last field mean "secure", whereas the s3 plugin
	// is using an older copy of minio (vendored), and this field used to mean "insecure".
	// See https://github.com/starkandwayne/shield/issues/230
	scalityClient, err := minio.NewV2(scal.Host, scal.AccessKey, scal.SecretKey, true)
	if err != nil {
		return nil, err
	}

	transport := http.DefaultTransport
	transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: scal.SkipSSLValidation}
	if scal.SOCKS5Proxy != "" {
		dialer, err := proxy.SOCKS5("tcp", scal.SOCKS5Proxy, nil, proxy.Direct)
		if err != nil {
			fmt.Fprintln(os.Stderr, "can't connect to the proxy:", err)
			os.Exit(1)
		}
		transport.(*http.Transport).Dial = dialer.Dial
	}

	scalityClient.SetCustomTransport(transport)

	return scalityClient, nil
}
