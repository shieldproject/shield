package main

import (
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	fmt "github.com/jhunt/go-ansi"
	"golang.org/x/oauth2"
	oauthgoogle "golang.org/x/oauth2/google"
	"google.golang.org/api/storage/v1"

	"github.com/starkandwayne/shield/plugin"
)

const (
	DefaultJsonKey = ""
	DefaultPrefix  = ""
)

func main() {
	p := GooglePlugin{
		Name:    "Google Cloud Storage Plugin",
		Author:  "Stark & Wayne",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
			Target: "no",
			Store:  "yes",
		},

		Example: `
{
  "json_key": "your-gcs-json-key",     # optional, Google Cloud JSON key
  "bucket":   "name-of-your-bucket",   # REQUIRED
  "prefix"    "/path/in/bucket"        # optional, where to store archives inside the bucket
}
`,
		Defaults: `
{
  # there are no defaults.
}
`,

		Fields: []plugin.Field{
			plugin.Field{
				Mode:  "store",
				Name:  "json_key",
				Type:  "string",
				Title: "Google Cloud JSON key",
				Help:  "Your Google Cloud Store JSON key, which should be available via the Google Cloud Platform web UI.",
			},
			plugin.Field{
				Mode:     "store",
				Name:     "bucket",
				Type:     "string",
				Title:    "Bucket Name",
				Help:     "Name of the bucket to store backup archives in.",
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

type GooglePlugin plugin.PluginInfo

type GoogleConnectionInfo struct {
	JsonKey string
	Bucket  string
	Prefix  string
}

func (p GooglePlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p GooglePlugin) Validate(endpoint plugin.ShieldEndpoint) error {
	var (
		s    string
		err  error
		fail bool
	)

	s, err = endpoint.StringValueDefault("json_key", DefaultJsonKey)
	if err != nil {
		fmt.Printf("@R{\u2717 json_key     %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 json_key}     (using Google Application Default Credentials)\n")
	} else {
		fmt.Printf("@G{\u2713 json_key}     @C{%s}\n", plugin.Redact(s))
	}

	s, err = endpoint.StringValue("bucket")
	if err != nil {
		fmt.Printf("@R{\u2717 bucket       %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 bucket}       @C{%s}\n", plugin.Redact(s))
	}

	s, err = endpoint.StringValueDefault("prefix", DefaultPrefix)
	if err != nil {
		fmt.Printf("@R{\u2717 prefix       %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 prefix}       (none)\n")
	} else {
		s = strings.TrimLeft(s, "/")
		fmt.Printf("@G{\u2713 prefix}       @C{%s}\n", s)
	}

	if fail {
		return fmt.Errorf("google: invalid configuration")
	}
	return nil
}

func (p GooglePlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	return plugin.UNIMPLEMENTED
}

func (p GooglePlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	return plugin.UNIMPLEMENTED
}

func (p GooglePlugin) Store(endpoint plugin.ShieldEndpoint) (string, int64, error) {
	gcs, err := getGoogleConnInfo(endpoint)
	if err != nil {
		return "", 0, err
	}

	client, err := gcs.Connect()
	if err != nil {
		return "", 0, err
	}

	path := gcs.genBackupPath()
	plugin.DEBUG("Storing data in %s", path)

	object, err := client.Objects.Insert(gcs.Bucket, &storage.Object{Name: path}).Media(os.Stdin).Do()
	if err != nil {
		return "", 0, err
	}

	return path, int64(object.Size), nil
}

func (p GooglePlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	gcs, err := getGoogleConnInfo(endpoint)
	if err != nil {
		return err
	}

	client, err := gcs.Connect()
	if err != nil {
		return err
	}

	res, err := client.Objects.Get(gcs.Bucket, file).Download()
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if _, err = io.Copy(os.Stdout, res.Body); err != nil {
		return err
	}

	return nil
}

func (p GooglePlugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	gcs, err := getGoogleConnInfo(endpoint)
	if err != nil {
		return err
	}

	client, err := gcs.Connect()
	if err != nil {
		return err
	}

	if err := client.Objects.Delete(gcs.Bucket, file).Do(); err != nil {
		return err
	}

	return nil
}

func getGoogleConnInfo(e plugin.ShieldEndpoint) (GoogleConnectionInfo, error) {
	jsonKey, err := e.StringValueDefault("json_key", DefaultJsonKey)
	if err != nil {
		return GoogleConnectionInfo{}, err
	}

	bucket, err := e.StringValue("bucket")
	if err != nil {
		return GoogleConnectionInfo{}, err
	}

	prefix, err := e.StringValueDefault("prefix", DefaultPrefix)
	if err != nil {
		return GoogleConnectionInfo{}, err
	}
	prefix = strings.TrimLeft(prefix, "/")

	return GoogleConnectionInfo{
		JsonKey: jsonKey,
		Bucket:  bucket,
		Prefix:  prefix,
	}, nil
}

func (gcs GoogleConnectionInfo) genBackupPath() string {
	t := time.Now()
	year, mon, day := t.Date()
	hour, min, sec := t.Clock()
	uuid := plugin.GenUUID()
	path := fmt.Sprintf("%s/%04d/%02d/%02d/%04d-%02d-%02d-%02d%02d%02d-%s", gcs.Prefix, year, mon, day, year, mon, day, hour, min, sec, uuid)
	path = strings.Replace(path, "//", "/", -1)
	return path
}

func (gcs GoogleConnectionInfo) Connect() (*storage.Service, error) {
	var err error
	var storageClient *http.Client

	if gcs.JsonKey != "" {
		storageJwtConf, err := oauthgoogle.JWTConfigFromJSON([]byte(gcs.JsonKey), storage.DevstorageFullControlScope)
		if err != nil {
			return nil, err
		}
		storageClient = storageJwtConf.Client(oauth2.NoContext)
	} else {
		storageClient, err = oauthgoogle.DefaultClient(oauth2.NoContext, storage.DevstorageFullControlScope)
		if err != nil {
			return nil, err
		}
	}

	storageService, err := storage.New(storageClient)
	if err != nil {
		return nil, err
	}

	return storageService, nil
}
