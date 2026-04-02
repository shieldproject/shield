package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	fmt "github.com/jhunt/go-ansi"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/shieldproject/shield/plugin"
	"github.com/shieldproject/shield/plugin/s3util"
)

const (
	DefaultS3Host              = "s3.amazonaws.com"
	DefaultRegion              = "us-east-1"
	DefaultSigVersion          = "4"
	DefaultPartSize            = "5M"
	DefaultSkipSSLValidation   = false
	DefaultUseInstanceProfiles = false
	credentialsEndpoint        = "http://169.254.169.254/latest/meta-data/iam/security-credentials"
	// IMDSv2 endpoints
	imdsTokenEndpoint = "http://169.254.169.254/latest/api/token"
	imdsTokenTTL      = "21600" // 6 hours in seconds
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

func isRedirectError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "PermanentRedirect") ||
		strings.Contains(msg, "Please send all future requests to this endpoint")
}

func main() {
	p := S3Plugin{
		Name:    "Amazon S3 Storage Plugin",
		Author:  "SHIELD Core Team",
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
  "s3_host"             : "s3.amazonaws.com",
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
	UsePathStyle        bool
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
		fmt.Printf("@R{\u2717 region                %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 region}                @C{%s} (default)\n", DefaultRegion)
	} else {
		fmt.Printf("@G{\u2713 region}                @C{%s}\n", s)
	}

	s, err = endpoint.StringValue("bucket")
	if err != nil {
		fmt.Printf("@R{\u2717 bucket                %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@R{\u2717 bucket                no s3 bucket specified}\n")
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
	c.UsePathStyle = true
	client, err := c.Connect()
	if err != nil {
		return "", 0, err
	}

	path := s3util.GenBackupPath(c.PathPrefix)
	plugin.Infof("storing backup archive\n"+
		"    at path   '%s'\n"+
		"    in bucket '%s'", path, c.Bucket)

	// Buffer stdin to a temp file so the body is seekable for AWS SDK v2
	// signature computation (PutObject requires seekable body for payload hash).
	tmpFile, err := os.CreateTemp("", "shield-s3-upload-*")
	if err != nil {
		return "", 0, fmt.Errorf("failed to create temp file for S3 upload: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	n, err := io.Copy(tmpFile, os.Stdin)
	if err != nil {
		return "", 0, fmt.Errorf("failed to buffer backup to temp file: %w", err)
	}
	if _, err := tmpFile.Seek(0, 0); err != nil {
		return "", 0, fmt.Errorf("failed to seek temp file: %w", err)
	}

	plugin.Infof("uploading %d bytes to s3", n)

	_, err = client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:        aws.String(c.Bucket),
		Key:           aws.String(path),
		Body:          tmpFile,
		ContentLength: &n,
	})
	if err != nil {
		if isRedirectError(err) {
			return "", 0, fmt.Errorf("s3: bucket redirect detected during Store — "+
				"set path_style or virtual-host style explicitly in plugin configuration: %w", err)
		}
		return "", 0, err
	}

	plugin.Infof("upload complete; uploaded %d bytes of data", n)
	return path, n, nil
}

func (p S3Plugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	e, err := getS3ConnInfo(endpoint)
	if err != nil {
		return err
	}

	plugin.Infof("connecting to s3...")
	e.UsePathStyle = true
	client, err := e.Connect()
	if err != nil {
		return err
	}

	plugin.Infof("retrieving backup archive\n"+
		"    from path '%s'\n"+
		"    in bucket '%s'", file, e.Bucket)

	ctx := context.Background()
	result, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(e.Bucket),
		Key:    aws.String(file),
	})
	if err != nil && isRedirectError(err) {
		e.UsePathStyle = false
		client, err2 := e.Connect()
		if err2 != nil {
			return err2
		}
		result, err = client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(e.Bucket),
			Key:    aws.String(file),
		})
	}
	if err != nil {
		return err
	}
	defer result.Body.Close()

	plugin.Infof("streaming backup archive to standard output")
	n, err := io.Copy(os.Stdout, result.Body)
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
	e.UsePathStyle = true
	client, err := e.Connect()
	if err != nil {
		return err
	}

	plugin.Infof("deleting backup archive\n"+
		"    at path   '%s'\n"+
		"    in bucket '%s'", file, e.Bucket)

	ctx := context.Background()
	_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(e.Bucket),
		Key:    aws.String(file),
	})
	if err != nil && isRedirectError(err) {
		e.UsePathStyle = false
		client, err2 := e.Connect()
		if err2 != nil {
			return err2
		}
		_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(e.Bucket),
			Key:    aws.String(file),
		})
	}
	if err != nil {
		return err
	}

	plugin.Infof("deleted backup archive\n"+
		"    at path   '%s'\n"+
		"    in bucket '%s'", file, e.Bucket)

	return nil
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

func (e s3Endpoint) Connect() (*s3.Client, error) {
	if e.SignatureVersion == 2 {
		plugin.Debugf("WARNING: signature_version=2 is not supported by AWS SDK v2; using SigV4 instead")
	}

	domain := e.Host
	if e.Port != "" && e.Port != "443" && e.Port != "80" {
		domain = e.Host + ":" + e.Port
	}
	endpoint := e.Protocol + "://" + domain

	cfg := s3util.S3Config{
		Region:             e.Region,
		AccessKeyID:        e.AccessKey,
		SecretAccessKey:    e.SecretKey,
		SessionToken:       e.Token,
		Endpoint:           endpoint,
		UsePathStyle:       e.UsePathStyle,
		InsecureSkipVerify: e.SkipSSLValidation,
		SOCKS5Proxy:        e.SOCKS5Proxy,
	}
	return s3util.NewClient(cfg)
}

func getInstanceProfileCredentials() (instanceProfileCredentials, error) {
	var creds instanceProfileCredentials

	// Step 1: Get IMDSv2 token
	tokenReq, err := http.NewRequest("PUT", imdsTokenEndpoint, nil)
	if err != nil {
		return creds, fmt.Errorf("failed to create token request: %v", err)
	}
	tokenReq.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", imdsTokenTTL)

	client := &http.Client{Timeout: 10 * time.Second}
	tokenResp, err := client.Do(tokenReq)
	if err != nil {
		return creds, fmt.Errorf("failed to get IMDSv2 token: %v", err)
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != 200 {
		return creds, fmt.Errorf("failed to get IMDSv2 token, status: %d", tokenResp.StatusCode)
	}

	tokenBytes, err := io.ReadAll(tokenResp.Body)
	if err != nil {
		return creds, fmt.Errorf("failed to read IMDSv2 token: %v", err)
	}
	token := strings.TrimSpace(string(tokenBytes))

	// Step 2: Get IAM role name using IMDSv2 token
	roleReq, err := http.NewRequest("GET", credentialsEndpoint, nil)
	if err != nil {
		return creds, fmt.Errorf("failed to create role request: %v", err)
	}
	roleReq.Header.Set("X-aws-ec2-metadata-token", token)

	roleResp, err := client.Do(roleReq)
	if err != nil {
		return creds, fmt.Errorf("failed to get IAM role: %v", err)
	}
	defer roleResp.Body.Close()

	if roleResp.StatusCode != 200 {
		return creds, fmt.Errorf("failed to get IAM role, status: %d", roleResp.StatusCode)
	}

	roleBytes, err := io.ReadAll(roleResp.Body)
	if err != nil {
		return creds, fmt.Errorf("failed to read IAM role: %v", err)
	}
	roleName := strings.TrimSpace(string(roleBytes))

	// Step 3: Get credentials using the role name and IMDSv2 token
	credReq, err := http.NewRequest("GET", credentialsEndpoint+"/"+roleName, nil)
	if err != nil {
		return creds, fmt.Errorf("failed to create credentials request: %v", err)
	}
	credReq.Header.Set("X-aws-ec2-metadata-token", token)

	credResp, err := client.Do(credReq)
	if err != nil {
		return creds, fmt.Errorf("failed to get credentials: %v", err)
	}
	defer credResp.Body.Close()

	if credResp.StatusCode != 200 {
		return creds, fmt.Errorf("failed to get credentials, status: %d", credResp.StatusCode)
	}

	credBytes, err := io.ReadAll(credResp.Body)
	if err != nil {
		return creds, fmt.Errorf("failed to read credentials: %v", err)
	}

	err = json.Unmarshal(credBytes, &creds)
	if err != nil {
		return creds, fmt.Errorf("failed to parse credentials JSON: %v", err)
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
