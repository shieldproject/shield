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
	DefaultPrefix              = ""
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
	re := regexp.MustCompile(`(?i)^(\d+)([mg])b?`)
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
				Help:    "The name of the region to operate in.  Some S3 work-alikes need this to be set explicitly.",
				Example: "us-east-1, US",
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
	SkipSSLValidation   bool
	UseInstanceProfiles bool
	AccessKey           string
	SecretKey           string
	Token               string
	Bucket              string
	Region              string
	PathPrefix          string
	SignatureVersion    int
	SOCKS5Proxy         string
	Port                string
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
		s    string
		err  error
		fail bool
	)

	s, err = endpoint.StringValueDefault("s3_host", DefaultS3Host)
	if err != nil {
		fmt.Printf("@R{\u2717 s3_host              %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 s3_host}              @C{%s}\n", s)
	}

	//BEGIN AUTH VALIDATION

	useInstanceProfiles, err := endpoint.BooleanValueDefault("use_instance_profile", DefaultUseInstanceProfiles)
	if err != nil {
		fmt.Printf("@R{\u2717 use_instance_profile  %s}\n", err)
		fail = true
	} else if useInstanceProfiles {
		fmt.Printf("@G{\u2713 use_instance_profile}  @C{yes}, AWS Instance Profiles @Y{WILL} be used\n")
	} else {
		fmt.Printf("@G{\u2713 use_instance_profile}  @C{no}, AWS Instance Profiles will @Y{NOT} be used\n")
	}
	//When using instance profiles, the key and secret are grabbed automatically.
	if !useInstanceProfiles {
		s, err = endpoint.StringValue("access_key_id")
		if err != nil {
			fmt.Printf("@R{\u2717 access_key_id        %s}\n", err)
			fail = true
		} else {
			fmt.Printf("@G{\u2713 access_key_id}        @C{%s}\n", plugin.Redact(s))
		}

		s, err = endpoint.StringValue("secret_access_key")
		if err != nil {
			fmt.Printf("@R{\u2717 secret_access_key    %s}\n", err)
			fail = true
		} else {
			fmt.Printf("@G{\u2713 secret_access_key}    @C{%s}\n", plugin.Redact(s))
		}
	}

	s, err = endpoint.StringValueDefault("s3_port", "")
	if err != nil {
		fmt.Printf("@R{\u2717 s3_port        %s}\n", err)
		fail = true
	} else {
		if s3Host, err := endpoint.StringValueDefault("s3_host", ""); s != "" && err == nil && s3Host == "" {
			fmt.Printf("@R{\u2717 s3_port        %s but s3_host cannot be empty}\n", s)
			fail = true
		} else {
			fmt.Printf("@G{\u2713 s3_port}        @C{%s}\n", s)
		}
	}
	//END AUTH VALIDATION

	s, err = endpoint.StringValue("bucket")
	if err != nil {
		fmt.Printf("@R{\u2717 bucket               %s}\n", err)
		fail = true
	} else if !validBucketName(s) {
		fmt.Printf("@R{\u2717 bucket               '%s' is an invalid bucket name (must be all lowercase)}\n", s)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 bucket}               @C{%s}\n", plugin.Redact(s))
	}

	s, err = endpoint.StringValueDefault("region", DefaultRegion)
	if err != nil {
		fmt.Printf("@R{\u2717 region               %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 region}               @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("prefix", DefaultPrefix)
	if err != nil {
		fmt.Printf("@R{\u2717 prefix               %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 prefix}               (none)\n")
	} else {
		s = strings.TrimLeft(s, "/")
		fmt.Printf("@G{\u2713 prefix}               @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("signature_version", DefaultSigVersion)
	if err != nil {
		fmt.Printf("@R{\u2717 signature_version    %s}\n", err)
		fail = true
	} else if !validSigVersion(s) {
		fmt.Printf("@R{\u2717 signature_version    Unexpected signature version '%s' found (expecting '2' or '4')}\n", s)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 signature_version}    @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("part_size", DefaultPartSize)
	if err != nil {
		fmt.Printf("@R{\u2717 part_size            %s}\n", err)
		fail = true
	} else if !validPartSize(s) {
		fmt.Printf("@R{\u2717 part_size            Invalid part size '%s'}\n", s)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 part_size}            @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("socks5_proxy", "")
	if err != nil {
		fmt.Printf("@R{\u2717 socks5_proxy         %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 socks5_proxy}         (no proxy will be used)\n")
	} else {
		fmt.Printf("@G{\u2713 socks5_proxy}         @C{%s}\n", s)
	}

	tf, err := endpoint.BooleanValueDefault("skip_ssl_validation", DefaultSkipSSLValidation)
	if err != nil {
		fmt.Printf("@R{\u2717 skip_ssl_validation  %s}\n", err)
		fail = true
	} else if tf {
		fmt.Printf("@G{\u2713 skip_ssl_validation}  @C{yes}, SSL will @Y{NOT} be validated\n")
	} else {
		fmt.Printf("@G{\u2713 skip_ssl_validation}  @C{no}, SSL @Y{WILL} be validated\n")
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

	client, err := c.Connect()
	if err != nil {
		return "", 0, err
	}

	path := c.genBackupPath()
	plugin.DEBUG("Storing data in %s", path)

	upload, err := client.NewUpload(path, nil)
	if err != nil {
		return "", 0, err
	}

	size, err := upload.Stream(os.Stdin, c.PartSize)
	if err != nil {
		return "", 0, err
	}

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

	client, err := e.Connect()
	if err != nil {
		return err
	}

	reader, err := client.Get(file)
	if err != nil {
		return err
	}
	_, err = io.Copy(os.Stdout, reader)
	return err
}

func (p S3Plugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	e, err := getS3ConnInfo(endpoint)
	if err != nil {
		return err
	}

	client, err := e.Connect()
	if err != nil {
		return err
	}

	return client.Delete(file)
}

func getS3ConnInfo(e plugin.ShieldEndpoint) (s3Endpoint, error) {
	var (
		key    string
		secret string
		token  string
	)
	host, err := e.StringValueDefault("s3_host", DefaultS3Host)
	if err != nil {
		return s3Endpoint{}, err
	}

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

	bucket, err := e.StringValue("bucket")
	if err != nil {
		return s3Endpoint{}, err
	}

	region, err := e.StringValueDefault("region", DefaultRegion)
	if err != nil {
		return s3Endpoint{}, err
	}

	prefix, err := e.StringValueDefault("prefix", DefaultPrefix)
	if err != nil {
		return s3Endpoint{}, err
	}
	prefix = strings.TrimLeft(prefix, "/")

	s, err := e.StringValueDefault("signature_version", DefaultSigVersion)
	if !validSigVersion(s) {
		return s3Endpoint{}, fmt.Errorf("Invalid `signature_version` specified (`%s`). Expected `2` or `4`", s)
	}
	sigVer := 4
	if s == "2" {
		sigVer = 2
	}

	s, err = e.StringValueDefault("part_size", DefaultPartSize)
	if !validPartSize(s) {
		return s3Endpoint{}, fmt.Errorf("Invalid `part_size` specified (`%s`).", s)
	}
	partSize := parsePartSize(s)

	proxy, err := e.StringValueDefault("socks5_proxy", "")
	if err != nil {
		return s3Endpoint{}, err
	}

	port, err := e.StringValueDefault("s3_port", "")
	if err != nil {
		return s3Endpoint{}, err
	}

	return s3Endpoint{
		Host:                host,
		SkipSSLValidation:   insecure_ssl,
		UseInstanceProfiles: useInstanceProfiles,
		AccessKey:           key,
		SecretKey:           secret,
		Token:               token,
		Bucket:              bucket,
		Region:              region,
		PathPrefix:          prefix,
		SignatureVersion:    sigVer,
		SOCKS5Proxy:         proxy,
		Port:                port,
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
	var protocol string
	host := e.Host

	if u, err := url.Parse(host); err == nil {
		protocol = u.Scheme
		host = u.Host
	}

	if e.Port != "" {
		host = host + ":" + e.Port
	}

	return s3.NewClient(&s3.Client{
		SignatureVersion:   e.SignatureVersion,
		AccessKeyID:        e.AccessKey,
		SecretAccessKey:    e.SecretKey,
		Token:              e.Token,
		Region:             e.Region,
		Domain:             host,
		Bucket:             e.Bucket,
		InsecureSkipVerify: e.SkipSSLValidation,
		SOCKS5Proxy:        e.SOCKS5Proxy,
		UsePathBuckets:     true,
		Protocol:           protocol,
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
