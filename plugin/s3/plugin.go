package main

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	fmt "github.com/jhunt/go-ansi"
	"github.com/jhunt/go-s3"

	"github.com/starkandwayne/shield/plugin"
)

const (
	DefaultS3Host              = "s3.amazonaws.com"
	DefaultRegion              = "us-east-1"
	DefaultSigVersion          = "4"
	DefaultPartSize            = "5M"
	DefaultSkipSSLValidation   = false
	DefaultUseInstanceProfiles = false
	credentialsEndpoint        = "http://169.254.169.254/latest/meta-data/iam/security-credentials"
)

func validSigVersion(v string) bool {
	return v == "2" || v == "4"
}

func parsePartSize(v string) int {
	re := regexp.MustCompile(`(?i)^(\d+)([mg])b?$`)
	m := re.FindStringSubmatch(v)
	if m == nil {
		return -1
	}
	n, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return -1
	}
	switch strings.ToLower(m[2]) {
	case "m":
		return int(n * 1024 * 1024)
	case "g":
		return int(n * 1024 * 1024 * 1024)
	default:
		return -1
	}
}

func validPartSize(v string) bool {
	return parsePartSize(v) >= 5*1024*1024
}

func validBucketName(v string) bool {
	ok, err := regexp.MatchString(`^[a-z0-9][a-z0-9\.\-]{1,61}[a-z0-9]$`, v)
	return ok && err == nil
}

func main() {
	p := S3Plugin{
		Name:    "Amazon S3 Storage Plugin",
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
  "part_size"           : "75m",                 # optional multipart upload part size
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
  "skip_ssl_validation" : false,
  "part_size"           : "5M"
}
`,
		Fields: []plugin.Field{
			plugin.Field{
				Mode:  "store",
				Name:  "use_instance_profile",
				Type:  "bool",
				Title: "Use Instance Profile",
				Help:  "Enable using AWS Instance Profiles instead of Access Key and Secret.",
			},
			plugin.Field{
				Mode:  "store",
				Name:  "access_key_id",
				Type:  "string",
				Title: "Access Key ID",
				Help:  "The Access Key ID to use when authenticating against S3.",
			},
			plugin.Field{
				Mode:  "store",
				Name:  "secret_access_key",
				Type:  "password",
				Title: "Secret Access Key",
				Help:  "The Secret Access Key to use when authenticating against S3.",
			},
			plugin.Field{
				Mode:    "store",
				Name:    "region",
				Type:    "string",
				Title:   "Region",
				Help:    "Name of the region this bucket exists in.",
				Default: DefaultRegion,
			},
			plugin.Field{
				Mode:     "store",
				Name:     "bucket",
				Type:     "string",
				Title:    "Bucket Name",
				Help:     "Name of the bucket to store backup archives in.",
				Example:  "my-aws-backups",
				Required: true,
			},
			plugin.Field{
				Mode:  "store",
				Name:  "prefix",
				Type:  "string",
				Title: "Bucket Path Prefix",
				Help:  "An optional sub-path of the bucket to use for storing archives.  By default, archives are stored in the root of the bucket.",
			},
			plugin.Field{
				Mode:    "store",
				Name:    "s3_host",
				Type:    "string",
				Title:   "S3 Host",
				Help:    "An alternative hostname or IP address for S3 work-alike implementations.  For AWS S3, leave this blank to auto-select the correct value.",
				Default: DefaultS3Host,
			},
			plugin.Field{
				Mode:  "store",
				Name:  "s3_port",
				Type:  "port",
				Title: "S3 Port",
				Help:  "An alternative TCP port to use for S3 work-alike implementations.  For AWS S3, leave this blank to auto-select the correct value.",
			},
			plugin.Field{
				Mode:    "store",
				Name:    "signature_version",
				Type:    "enum",
				Enum:    []string{"4", "2"},
				Title:   "AWS Signature Version",
				Help:    "Specify an alternate signature version.  For AWS S3, leave this blank to auto-select the correct value.",
				Default: DefaultSigVersion,
			},
			plugin.Field{
				Mode:    "store",
				Name:    "part_size",
				Type:    "string",
				Title:   "Multipart Upload Part Size",
				Help:    "How big should the individual parts of the backup upload be?  This must be at least 5M.",
				Example: "100MB, 64M, etc.",
				Default: DefaultPartSize,
			},
			plugin.Field{
				Mode:  "store",
				Name:  "socks5_proxy",
				Type:  "string",
				Title: "SOCKS5 Proxy",
				Help:  "The host:port address of a SOCKS5 proxy to relay HTTP through when accessing S3 work-alikes.",
			},
			plugin.Field{
				Mode:  "store",
				Name:  "skip_ssl_validation",
				Type:  "bool",
				Title: "Skip SSL Validation",
				Help:  "If your S3 work-alike certificate is invalid, expired, or signed by an unknown Certificate Authority, you can disable SSL validation.  This is not recommended from a security standpoint, however.",
			},
		},
	}

	plugin.Run(p)
}

type S3Plugin plugin.PluginInfo

type s3Endpoint struct {
	Host                string
	Port                string
	Protocol            string
	SkipSSLValidation   bool
	UseInstanceProfiles bool
	AccessKey           string
	SecretKey           string
	Token               string
	Region              string
	Bucket              string
	PathPrefix          string
	SignatureVersion    int
	SOCKS5Proxy         string
	PartSize            int
}

type instanceProfileCredentials struct {
	Key    string `json:"AccessKeyId"`
	Secret string `json:"SecretAccessKey"`
	Token  string `json:"Token"`
}

func (p S3Plugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p S3Plugin) Validate(endpoint plugin.ShieldEndpoint) error {
	var (
		s3_host, scheme, host, port string

		err  error
		fail bool
	)

	s, err := endpoint.StringValueDefault("s3_host", DefaultS3Host)
	if err != nil {
		fmt.Printf("@R{\u2717 s3_host               %s}\n", err)
		fail = true
	} else {
		s3_host = s /* save for s3_port reporting */
		scheme, host, port = parse(s)
		if host != s3_host {
			fmt.Printf("@G{\u2713 s3_host}               @C{%s} via @C{%s} (from @C{%s})\n", host, scheme, s)
		} else {
			fmt.Printf("@G{\u2713 s3_host}               @C{%s} via @C{%s}\n", s, scheme)
		}
	}

	s, err = endpoint.StringValueDefault("s3_port", "")
	if err != nil {
		fmt.Printf("@R{\u2717 s3_port               %s}\n", err)
		fail = true
	} else {
		if h, err := endpoint.StringValueDefault("s3_host", ""); s != "" && err == nil && h == "" {
			fmt.Printf("@R{\u2717 s3_port               %s but s3_host cannot be empty}\n", s)
			fail = true
		} else if s == "" {
			fmt.Printf("@G{\u2713 s3_port}               @C{%s} (from s3_host: @C{%s})\n", port, h)
		} else {
			fmt.Printf("@G{\u2713 s3_port}               @C{%s} (manually overridden)\n", s)
		}
	}

	useInstanceProfiles, err := endpoint.BooleanValueDefault("use_instance_profile", DefaultUseInstanceProfiles)
	if err != nil {
		fmt.Printf("@R{\u2717 use_instance_profile   %s}\n", err)
		fail = true
	} else if useInstanceProfiles {
		fmt.Printf("@G{\u2713 use_instance_profile}  @C{yes}, AWS Instance Profiles @Y{WILL} be used\n")
	} else {
		fmt.Printf("@G{\u2713 use_instance_profile}  @C{no}, AWS Instance Profiles will @Y{NOT} be used\n")
	}

	if !useInstanceProfiles {
		s, err = endpoint.StringValue("access_key_id")
		if err != nil {
			fmt.Printf("@R{\u2717 access_key_id         %s}\n", err)
			fail = true
		} else {
			fmt.Printf("@G{\u2713 access_key_id}         @C{%s}\n", plugin.Redact(s))
		}

		s, err = endpoint.StringValue("secret_access_key")
		if err != nil {
			fmt.Printf("@R{\u2717 secret_access_key     %s}\n", err)
			fail = true
		} else {
			fmt.Printf("@G{\u2713 secret_access_key}     @C{%s}\n", plugin.Redact(s))
		}
	}

	s, err = endpoint.StringValueDefault("region", "")
	if err != nil {
		fmt.Printf("@R{\u2717 bucket                %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 bucket}                @C{%s} (default)\n", DefaultRegion)
	} else {
		fmt.Printf("@G{\u2713 bucket}                @C{%s}\n", s)
	}

	s, err = endpoint.StringValue("bucket")
	if err != nil {
		fmt.Printf("@R{\u2717 bucket                %s}\n", err)
		fail = true
	} else if !validBucketName(s) {
		fmt.Printf("@R{\u2717 bucket                '%s' is an invalid bucket name (must be all lowercase)}\n", s)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 bucket}                @C{%s}\n", plugin.Redact(s))
	}

	s, err = endpoint.StringValueDefault("prefix", "")
	if err != nil {
		fmt.Printf("@R{\u2717 prefix                %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 prefix}                (none)\n")
	} else {
		s = strings.TrimLeft(s, "/")
		fmt.Printf("@G{\u2713 prefix}                @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("signature_version", DefaultSigVersion)
	if err != nil {
		fmt.Printf("@R{\u2717 signature_version     %s}\n", err)
		fail = true
	} else if !validSigVersion(s) {
		fmt.Printf("@R{\u2717 signature_version     Unexpected signature version '%s' found (expecting '2' or '4')}\n", s)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 signature_version}     @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("part_size", DefaultPartSize)
	if err != nil {
		fmt.Printf("@R{\u2717 part_size             %s}\n", err)
		fail = true
	} else if !validPartSize(s) {
		fmt.Printf("@R{\u2717 part_size             Invalid part size '%s'}\n", s)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 part_size}             @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("socks5_proxy", "")
	if err != nil {
		fmt.Printf("@R{\u2717 socks5_proxy          %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 socks5_proxy}          (no proxy will be used)\n")
	} else {
		fmt.Printf("@G{\u2713 socks5_proxy}          @C{%s}\n", s)
	}

	tf, err := endpoint.BooleanValueDefault("skip_ssl_validation", DefaultSkipSSLValidation)
	if err != nil {
		fmt.Printf("@R{\u2717 skip_ssl_validation   %s}\n", err)
		fail = true
	} else if tf {
		fmt.Printf("@G{\u2713 skip_ssl_validation}   @C{yes}, SSL will @Y{NOT} be validated\n")
	} else {
		fmt.Printf("@G{\u2713 skip_ssl_validation}   @C{no}, SSL @Y{WILL} be validated\n")
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

func (p S3Plugin) Store(endpoint plugin.ShieldEndpoint) (string, int64, error) {
	c, err := getS3ConnInfo(endpoint)
	if err != nil {
		return "", 0, err
	}

	plugin.Infof("connecting to s3...")
	client, err := c.Connect()
	if err != nil {
		return "", 0, err
	}

	path := c.genBackupPath()
	plugin.Infof("storing backup archive\n"+
		"    at path   '%s'\n"+
		"    in bucket '%s'", path, c.Bucket)

	upload, err := client.NewUpload(path, nil)
	if err != nil {
		return "", 0, err
	}

	plugin.Infof("streaming standard input to s3 in %d-byte blocks", c.PartSize)
	size, err := upload.Stream(os.Stdin, c.PartSize)
	if err != nil {
		return "", 0, err
	}

	plugin.Infof("upload complete; uploaded %d bytes of data", size)
	err = upload.Done()
	if err != nil {
		return "", 0, err
	}

	return path, size, nil
}

func (p S3Plugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	e, err := getS3ConnInfo(endpoint)
	if err != nil {
		return err
	}

	plugin.Infof("connecting to s3...")
	c, err := e.Connect()
	if err != nil {
		return err
	}

	plugin.Infof("retrieving backup archive\n"+
		"    from path '%s\n"+
		"    in bucket '%s'", file, c.Bucket)
	reader, err := c.Get(file)
	if err != nil {
		return err
	}

	plugin.Infof("streaming backup archive to standard output")
	n, err := io.Copy(os.Stdout, reader)
	if err != nil {
		return err
	}

	plugin.Infof("retrieved %d bytes of data", n)
	return nil
}

func (p S3Plugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	e, err := getS3ConnInfo(endpoint)
	if err != nil {
		return err
	}

	plugin.Infof("connecting to s3...")
	c, err := e.Connect()
	if err != nil {
		return err
	}

	plugin.Infof("deleting backup archive\n"+
		"    at path   '%s'\n"+
		"    in bucket '%s'", file, c.Bucket)
	return c.Delete(file)
}

func getS3ConnInfo(e plugin.ShieldEndpoint) (s3Endpoint, error) {
	var (
		key    string
		secret string
		token  string
	)
	s3_host, err := e.StringValueDefault("s3_host", DefaultS3Host)
	if err != nil {
		return s3Endpoint{}, err
	}
	scheme, host, port := parse(s3_host)

	insecure_ssl, err := e.BooleanValueDefault("skip_ssl_validation", DefaultSkipSSLValidation)
	if err != nil {
		return s3Endpoint{}, err
	}

	useInstanceProfiles, err := e.BooleanValueDefault("use_instance_profile", DefaultUseInstanceProfiles)
	if err != nil {
		return s3Endpoint{}, err
	}

	if !useInstanceProfiles {
		key, err = e.StringValue("access_key_id")
		if err != nil {
			return s3Endpoint{}, err
		}

		secret, err = e.StringValue("secret_access_key")
		if err != nil {
			return s3Endpoint{}, err
		}
	} else {
		instanceProfileCreds, err := getInstanceProfileCredentials()
		if err != nil {
			return s3Endpoint{}, err
		}
		key = instanceProfileCreds.Key
		secret = instanceProfileCreds.Secret
		token = instanceProfileCreds.Token
	}

	region, err := e.StringValueDefault("region", DefaultRegion)
	if err != nil {
		return s3Endpoint{}, err
	}

	bucket, err := e.StringValue("bucket")
	if err != nil {
		return s3Endpoint{}, err
	}

	prefix, err := e.StringValueDefault("prefix", "")
	if err != nil {
		return s3Endpoint{}, err
	}
	prefix = strings.TrimLeft(prefix, "/")

	s, err := e.StringValueDefault("signature_version", DefaultSigVersion)
	if err != nil {
		return s3Endpoint{}, err
	}
	if !validSigVersion(s) {
		return s3Endpoint{}, fmt.Errorf("Invalid `signature_version` specified (`%s`). Expected `2` or `4`", s)
	}
	sigVer := 4
	if s == "2" {
		sigVer = 2
	}

	s, err = e.StringValueDefault("part_size", DefaultPartSize)
	if err != nil {
		return s3Endpoint{}, err
	}
	if !validPartSize(s) {
		return s3Endpoint{}, fmt.Errorf("Invalid `part_size` specified (`%s`).", s)
	}
	partSize := parsePartSize(s)

	proxy, err := e.StringValueDefault("socks5_proxy", "")
	if err != nil {
		return s3Endpoint{}, err
	}

	override, err := e.StringValueDefault("s3_port", "")
	if err != nil {
		return s3Endpoint{}, err
	}
	if override != "" {
		port = override
	}

	return s3Endpoint{
		Host:                host,
		Port:                port,
		Protocol:            scheme,
		SkipSSLValidation:   insecure_ssl,
		UseInstanceProfiles: useInstanceProfiles,
		AccessKey:           key,
		SecretKey:           secret,
		Token:               token,
		Region:              region,
		Bucket:              bucket,
		PathPrefix:          prefix,
		SignatureVersion:    sigVer,
		SOCKS5Proxy:         proxy,
		PartSize:            partSize,
	}, nil
}

func (e s3Endpoint) genBackupPath() string {
	t := time.Now()
	year, mon, day := t.Date()
	hour, min, sec := t.Clock()
	uuid := plugin.GenUUID()
	path := fmt.Sprintf("%s/%04d/%02d/%02d/%04d-%02d-%02d-%02d%02d%02d-%s", e.PathPrefix, year, mon, day, year, mon, day, hour, min, sec, uuid)
	// Remove double slashes
	path = strings.Replace(path, "//", "/", -1)
	return path
}

func (e s3Endpoint) Connect() (*s3.Client, error) {
	return s3.NewClient(&s3.Client{
		Protocol: e.Protocol,
		Domain:   e.Host + ":" + e.Port,

		SignatureVersion: e.SignatureVersion,
		AccessKeyID:      e.AccessKey,
		SecretAccessKey:  e.SecretKey,
		Token:            e.Token,

		Region: e.Region,
		Bucket: e.Bucket,

		InsecureSkipVerify: e.SkipSSLValidation,
		SOCKS5Proxy:        e.SOCKS5Proxy,
		UsePathBuckets:     true,
		/* FIXME: CA Certs */
	})
}

func getInstanceProfileCredentials() (instanceProfileCredentials, error) {
	response, connectErr := http.Get(fmt.Sprintf("%s/", credentialsEndpoint))
	if connectErr != nil {
		return instanceProfileCredentials{}, connectErr
	} else if response.StatusCode != 200 {
		return instanceProfileCredentials{}, errors.New(fmt.Sprintf("Connection request to %s/ failed with code %d", credentialsEndpoint, response.StatusCode))
	}

	body, readErr := ioutil.ReadAll(response.Body)
	if readErr != nil {
		return instanceProfileCredentials{}, readErr
	}
	role := string(body)
	response.Body.Close()

	var creds instanceProfileCredentials
	response, connectErr = http.Get(fmt.Sprintf("%s/%s", credentialsEndpoint, role))
	if connectErr != nil {
		return instanceProfileCredentials{}, connectErr
	} else if response.StatusCode != 200 {
		return instanceProfileCredentials{}, errors.New(fmt.Sprintf("Connection request to %s/%s failed with code %d", credentialsEndpoint, role, response.StatusCode))
	}
	defer response.Body.Close()

	body, readErr = ioutil.ReadAll(response.Body)
	if readErr != nil {
		return instanceProfileCredentials{}, readErr
	}

	unmarshallErr := json.Unmarshal(body, &creds)
	if unmarshallErr != nil {
		return instanceProfileCredentials{}, unmarshallErr
	}

	return creds, nil
}

func parse(host string) (string, string, string) {
	if u, err := url.Parse(host); err == nil && u.Host != "" {
		port := u.Port()
		if port == "" {
			if u.Scheme == "https" {
				port = "443"
			} else {
				port = "80"
			}
		}
		return u.Scheme, u.Hostname(), port
	}

	return "https", host, "443"
}
