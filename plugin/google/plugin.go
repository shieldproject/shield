// The `google` plugin for SHIELD is intended to be a back-end storage
// plugin, wrapping Google's Cloud Storage service.
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
// how to connect to Google Cloud Storage, and where to place/retrieve the data once connected.
// Your endpoint JSON should look something like this:
//
//    {
//        "json_key": "your-google-cloud-json-key",
//        "bucket":   "name-of-your-bucket",
//        "prefix":   "/path/inside/bucket/to/place/backup/data"
//    }
//
// Default Configuration
//
// `json_key` is only required if you are not running the plugin inside a Google Compute Engine VM
// with `devstorage.full_control` service scope, otherwise Google Application Default Credentials
// will be used (see https://developers.google.com/identity/protocols/application-default-credentials).
//
// `prefix` will default to the empty string, and backups will be placed in the
// root of the bucket.
//
// STORE DETAILS
//
// When storing data, this plugin connects to the Google Cloud Storage service, and uploads
// the data into the specified bucket, using a filename with the following format:
//
//    <prefix>/<YYYY>/<MM>/<DD>/<YYYY>-<MM>-<DD><HH-mm-SS>-<UUID>
//
// Upon successful storage, the plugin then returns this filename to SHIELD to use
// as the `store_key` when the data needs to be retrieved, or purged.
//
// RETRIEVE DETAILS
//
// When retrieving data, this plugin connects to the Google Cloud Storage service, and retrieves the data
// located in the specified bucket, identified by the `store_key` provided by SHIELD.
//
// PURGE DETAILS
//
// When purging data, this plugin connects to the Google Cloud Storage service, and deletes the data
// located in the specified bucket, identified by the `store_key` provided by SHIELD.
//
// DEPENDENCIES
//
// None.
//
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/starkandwayne/goutils/ansi"
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
		ansi.Printf("@R{\u2717 json_key     %s}\n", err)
		fail = true
	} else if s == "" {
		ansi.Printf("@G{\u2713 json_key}     (using Google Application Default Credentials)\n")
	} else {
		ansi.Printf("@G{\u2713 json_key}     @C{%s}\n", s)
	}

	s, err = endpoint.StringValue("bucket")
	if err != nil {
		ansi.Printf("@R{\u2717 bucket       %s}\n", err)
		fail = true
	} else {
		ansi.Printf("@G{\u2713 bucket}       @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("prefix", DefaultPrefix)
	if err != nil {
		ansi.Printf("@R{\u2717 prefix       %s}\n", err)
		fail = true
	} else if s == "" {
		ansi.Printf("@G{\u2713 prefix}       (none)\n")
	} else {
		s = strings.TrimLeft(s, "/")
		ansi.Printf("@G{\u2713 prefix}       @C{%s}\n", s)
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

func (p GooglePlugin) Store(endpoint plugin.ShieldEndpoint) (string, error) {
	gcs, err := getGoogleConnInfo(endpoint)
	if err != nil {
		return "", err
	}

	client, err := gcs.Connect()
	if err != nil {
		return "", err
	}

	path := gcs.genBackupPath()
	plugin.DEBUG("Storing data in %s", path)

	if _, err := client.Objects.Insert(gcs.Bucket, &storage.Object{Name: path}).Media(os.Stdin).Do(); err != nil {
		return "", err
	}

	return path, nil
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
