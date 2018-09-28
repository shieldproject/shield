package main

import (
	"context"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	fmt "github.com/jhunt/go-ansi"
	"github.com/kurin/blazer/b2"

	"github.com/starkandwayne/shield/plugin"
)

const DefaultPrefix = ""

func validBucketName(v string) bool {
	ok, err := regexp.MatchString(`^[a-z0-9][a-z0-9\.\-]{1,61}[a-z0-9]$`, v)
	return ok && err == nil
}

func main() {
	p := BackblazePlugin{
		Name:    "Backblaze Storage Plugin",
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

  "prefix"              : "/path/in/bucket",     # where to store archives, inside the bucket
}
`,
		Defaults: `
{
}
`,
		Fields: []plugin.Field{
			plugin.Field{
				Mode:  "store",
				Name:  "access_key_id",
				Type:  "string",
				Title: "Access Key ID",
				Help:  "The Access Key ID to use when authenticating against B2.",
			},
			plugin.Field{
				Mode:  "store",
				Name:  "secret_access_key",
				Type:  "password",
				Title: "Secret Access Key",
				Help:  "The Secret Access Key to use when authenticating against B2.",
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
		},
	}

	plugin.Run(p)
}

type BackblazePlugin plugin.PluginInfo

type backblazeEndpoint struct {
	AccessKey  string
	SecretKey  string
	PathPrefix string
	Bucket     string
}

func (p BackblazePlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p BackblazePlugin) Validate(endpoint plugin.ShieldEndpoint) error {
	var (
		s    string
		err  error
		fail bool
	)

	//BEGIN AUTH VALIDATION
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

	if fail {
		return fmt.Errorf("b2: invalid configuration")
	}
	return nil
}

func (p BackblazePlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	return plugin.UNIMPLEMENTED
}

func (p BackblazePlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	return plugin.UNIMPLEMENTED
}

func (p BackblazePlugin) Store(endpoint plugin.ShieldEndpoint) (string, int64, error) {
	c, err := getB2ConnInfo(endpoint)
	if err != nil {
		return "", 0, err
	}

	client, err := c.Connect()
	if err != nil {
		return "", 0, err
	}

	path := c.genBackupPath()
	plugin.DEBUG("Storing data in %s", path)

	ctx := context.Background()
	bucket, err := client.Bucket(ctx, c.Bucket)
	if err != nil {
		return "", 0, err
	}
	obj := bucket.Object(strings.TrimPrefix(path, "/"))
	w := obj.NewWriter(ctx)
	io.Copy(w, os.Stdin)

	if err != nil {
		return "", 0, err
	}

	w.Close()
	if err != nil {
		return "", 0, err
	}

	var size int64
	size = 1024

	return path, size, nil
}

func (p BackblazePlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	e, err := getB2ConnInfo(endpoint)
	if err != nil {
		return err
	}

	client, err := e.Connect()
	if err != nil {
		return err
	}

	ctx := context.Background()
	bucket, err := client.Bucket(ctx, e.Bucket)
	if err != nil {
		return err
	}
	obj := bucket.Object(strings.TrimPrefix(file, "/"))
	reader := obj.NewReader(ctx)
	if err != nil {
		return err
	}

	_, err = io.Copy(os.Stdout, reader)
	return err
}

func (p BackblazePlugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	e, err := getB2ConnInfo(endpoint)
	if err != nil {
		return err
	}

	client, err := e.Connect()
	if err != nil {
		return err
	}
	ctx := context.Background()
	bucket, err := client.Bucket(ctx, e.Bucket)
	if err != nil {
		return err
	}
	obj := bucket.Object(strings.TrimPrefix(file, "/"))

	return obj.Delete(ctx)
}

func getB2ConnInfo(e plugin.ShieldEndpoint) (backblazeEndpoint, error) {
	var (
		key    string
		secret string
		err    error
	)

	key, err = e.StringValue("access_key_id")
	if err != nil {
		return backblazeEndpoint{}, err
	}

	secret, err = e.StringValue("secret_access_key")
	if err != nil {
		return backblazeEndpoint{}, err
	}

	bucket, err := e.StringValue("bucket")
	if err != nil {
		return backblazeEndpoint{}, err
	}

	prefix, err := e.StringValueDefault("prefix", DefaultPrefix)
	if err != nil {
		return backblazeEndpoint{}, err
	}
	prefix = strings.TrimLeft(prefix, "/")

	return backblazeEndpoint{
		AccessKey:  key,
		SecretKey:  secret,
		PathPrefix: prefix,
		Bucket:     bucket,
	}, nil
}

func (e backblazeEndpoint) genBackupPath() string {
	t := time.Now()
	year, mon, day := t.Date()
	hour, min, sec := t.Clock()
	uuid := plugin.GenUUID()
	path := fmt.Sprintf("%s/%04d/%02d/%02d/%04d-%02d-%02d-%02d%02d%02d-%s", e.PathPrefix, year, mon, day, year, mon, day, hour, min, sec, uuid)
	// Remove double slashes
	path = strings.Replace(path, "//", "/", -1)
	return path
}

func (e backblazeEndpoint) Connect() (*b2.Client, error) {
	ctx := context.Background()
	return b2.NewClient(ctx, e.AccessKey, e.SecretKey)
}
