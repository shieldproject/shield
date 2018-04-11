// The `swift` plugin for SHIELD is intended to be a back-end storage
// plugin, wrapping OpenStack Swift.
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
//        "auth_url":      "host",
//        "project_name":  "openstack-project",
//        "username":      "your-username",
//        "password":      "secret-access-key",
//        "container":     "bucket-name",
//        "prefix":        "/path/inside/bucket/to/place/backup/data",
//        "debug":         false
//    }
//
// Default Configuration
//
//    {
//        "prefix" : "",
//        "debug"  : false
//    }
//
// STORE DETAILS
//
// When storing data, this plugin connects to the Swift service, and uploads the data
// into the specified container, using a path/filename with the following format:
//
//    <prefix>/<YYYY>/<MM>/<DD>/<HH-mm-SS>-<UUID>
//
// Upon successful storage, the plugin then returns this filename to SHIELD to use
// as the `store_key` when the data needs to be retrieved, or purged.
//
// RETRIEVE DETAILS
//
// When retrieving data, this plugin connects to the Swift service, and retrieves the data
// located in the specified container, identified by the `store_key` provided by SHIELD.
//
// PURGE DETAILS
//
// When purging data, this plugin connects to the Swift service, and deletes the data
// located in the specified container, identified by the `store_key` provided by SHIELD.
//
// DEPENDENCIES
//
// None.
//
package main

// https://github.com/openstack/golang-client/blob/master/examples/objectstorage/objectstorage.go

import (
	"bufio"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"git.openstack.org/openstack/golang-client/objectstorage/v1"
	"git.openstack.org/openstack/golang-client/openstack"
	fmt "github.com/jhunt/go-ansi"

	"github.com/starkandwayne/shield/plugin"
)

const (
	defaultDebug  = false
	defaultPrefix = ""
)

func main() {
	p := SwiftPlugin{
		Name:    "OpenStack Swift Storage Plugin",
		Author:  "Stark & Wayne",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
			Target: "no",
			Store:  "yes",
		},
		Example: `
{
  "auth_url":      "https://identity.api.rackspacecloud.com/v2.0",
  "project_name":  "openstack-project",
  "username":      "your-username",
  "password":      "secret-access-key",
  "container":     "bucket-name",
  "prefix":        "/path/inside/bucket/to/place/backup/data",
  "debug":         false
}
`,
		Defaults: `
{
  "prefix":        "",
  "debug":         false
}
`,

		Fields: []plugin.Field{
			plugin.Field{
				Mode:     "store",
				Name:     "auth_url",
				Type:     "string",
				Title:    "Authentication URL",
				Help:     "The URL of the authentication API",
				Example:  "https://identity.api.rackspacecloud.com/v2.0",
				Required: true,
			},
			plugin.Field{
				Mode:     "store",
				Name:     "project_name",
				Type:     "string",
				Title:    "Project Name",
				Help:     "Name of the openstack project",
				Required: true,
			},
			plugin.Field{
				Mode:     "store",
				Name:     "username",
				Type:     "string",
				Title:    "Username",
				Help:     "The username used to authenticate to swift",
				Required: true,
			},
			plugin.Field{
				Mode:     "store",
				Name:     "password",
				Type:     "password",
				Title:    "Password",
				Help:     "The password used to authenticate to swift",
				Required: true,
			},
			plugin.Field{
				Mode:     "store",
				Name:     "container",
				Type:     "string",
				Title:    "Container",
				Help:     "Name of the container to store backup archives in.",
				Required: true,
			},
			plugin.Field{
				Mode:  "store",
				Name:  "prefix",
				Type:  "string",
				Title: "Bucket Path Prefix",
				Help:  "An optional sub-path of the container to use for storing archives.  By default, archives are stored in the root of the container.",
			},
		},
	}

	plugin.Run(p)
}

type SwiftPlugin plugin.PluginInfo

type SwiftConnectionInfo struct {
	AuthURL     string
	ProjectName string
	Username    string
	Password    string
	Container   string
	PathPrefix  string
	Debug       bool
}

func (p SwiftPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p SwiftPlugin) Validate(endpoint plugin.ShieldEndpoint) (err error) {
	var s string
	var fail bool

	requiredConfig := []string{"auth_url", "project_name", "username", "password", "container"}
	for _, reqConfig := range requiredConfig {
		s, err = endpoint.StringValue(reqConfig)
		if err != nil {
			fmt.Printf("@R{\u2717 %s   %s}\n", reqConfig, err)
			fail = true
		} else {
			if reqConfig == "auth_url" || reqConfig == "project_name" {
				fmt.Printf("@G{\u2713 %s}   @C{%s}\n", reqConfig, s)
			} else {
				fmt.Printf("@G{\u2713 %s}   @C{%s}\n", reqConfig, plugin.Redact(s))
			}
		}
	}
	if fail {
		return fmt.Errorf("swift: invalid configuration")
	}
	return
}

func (p SwiftPlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	return plugin.UNIMPLEMENTED
}

func (p SwiftPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	return plugin.UNIMPLEMENTED
}

func (p SwiftPlugin) Store(endpoint plugin.ShieldEndpoint) (string, int64, error) {
	swift, err := getConnInfo(endpoint)
	if err != nil {
		return "", 0, err
	}
	openstack.Debug = &swift.Debug

	baseURL, session, err := swift.Connect()
	if err != nil {
		return "", 0, err
	}

	path := swift.genBackupPath()
	plugin.DEBUG("Storing data in %s", path)

	r := bufio.NewReader(os.Stdin)
	contents, err := ioutil.ReadAll(r)
	if err != nil {
		return "", 0, err
	}

	headers := http.Header{}
	url := baseURL + "/" + swift.Container + "/" + path
	err = objectstorage.PutObject(session, &contents, url, headers)
	if err != nil {
		return "", 0, err
	}

	return path, int64(len(contents)), nil
}

func (p SwiftPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) (err error) {
	swift, err := getConnInfo(endpoint)
	if err != nil {
		return
	}
	openstack.Debug = &swift.Debug
	baseURL, session, err := swift.Connect()
	if err != nil {
		return
	}

	url := baseURL + "/" + swift.Container + "/" + file
	_, contents, err := objectstorage.GetObject(session, url)
	if err != nil {
		return
	}

	_, err = os.Stdout.Write(contents)
	return
}

func (p SwiftPlugin) Purge(endpoint plugin.ShieldEndpoint, file string) (err error) {
	swift, err := getConnInfo(endpoint)
	if err != nil {
		return
	}
	baseURL, session, err := swift.Connect()
	if err != nil {
		return
	}

	url := baseURL + "/" + swift.Container + "/" + file
	err = objectstorage.DeleteObject(session, url)
	return
}

func getConnInfo(e plugin.ShieldEndpoint) (info *SwiftConnectionInfo, err error) {
	info = &SwiftConnectionInfo{}
	info.AuthURL, err = e.StringValue("auth_url")
	if err != nil {
		return
	}

	info.ProjectName, err = e.StringValue("project_name")
	if err != nil {
		return
	}

	info.Username, err = e.StringValue("username")
	if err != nil {
		return
	}

	info.Password, err = e.StringValue("password")
	if err != nil {
		return
	}

	info.Container, err = e.StringValue("container")
	if err != nil {
		return
	}

	info.PathPrefix, err = e.StringValueDefault("prefix", defaultPrefix)
	if err != nil {
		return
	}
	info.PathPrefix = strings.TrimLeft(info.PathPrefix, "/")

	info.Debug, err = e.BooleanValueDefault("debug", defaultDebug)
	if err != nil {
		return
	}

	return
}

func (info SwiftConnectionInfo) genBackupPath() string {
	t := time.Now()
	year, mon, day := t.Date()
	hour, min, sec := t.Clock()
	uuid := plugin.GenUUID()
	path := fmt.Sprintf("%s/%04d/%02d/%02d/%04d-%02d-%02d-%02d%02d%02d-%s", info.PathPrefix, year, mon, day, year, mon, day, hour, min, sec, uuid)
	// Remove double slashes
	path = strings.Replace(path, "//", "/", -1)
	return path
}

func (swift SwiftConnectionInfo) Connect() (baseURL string, session *openstack.Session, err error) {
	creds := openstack.AuthOpts{
		AuthUrl:     swift.AuthURL,
		ProjectName: swift.ProjectName,
		Username:    swift.Username,
		Password:    swift.Password,
	}
	auth, err := openstack.DoAuthRequest(creds)
	if err != nil {
		return
	}
	if !auth.GetExpiration().After(time.Now()) {
		return "", nil, fmt.Errorf("There was an error. The auth token has an invalid expiration.")
	}

	// Find the endpoint for object storage.
	baseURL, err = auth.GetEndpoint("object-store", "")
	if baseURL == "" || err != nil {
		return "", nil, fmt.Errorf("object-store url not found during authentication")
	}

	// Make a new client with these creds
	session, err = openstack.NewSession(nil, auth, nil)
	if err != nil {
		return "", nil, fmt.Errorf("Error crating new Session: %v", err)
	}

	_, err = objectstorage.GetAccountMeta(session, baseURL)
	if err != nil {
		return "", nil, fmt.Errorf("There was an error getting account metadata: %v", err)
	}

	return
}
