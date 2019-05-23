package main

// https://github.com/openstack/golang-client/blob/master/examples/objectstorage/objectstorage.go

import (
	"bufio"
	"io/ioutil"
	"os"
	"strings"
	"time"

	fmt "github.com/jhunt/go-ansi"

	"github.com/ncw/swift"

	"github.com/starkandwayne/shield/plugin"
)

const (
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
}
`,
		Defaults: `
{
  "prefix":        "",
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
				Help:     "Name of the openstack project/tenant (v2 auth only)",
				Required: false,
			},
			plugin.Field{
				Mode:     "store",
				Name:     "domain",
				Type:     "string",
				Title:    "Domain",
				Help:     "Name of the openstack domain (v3 auth only)",
				Required: false,
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
	ProjectName string // v2 auth only
	Domain      string // v3 auth only
	Username    string
	Password    string
	Container   string
	PathPrefix  string
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

	conn, err := swift.Connect()
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

	if err := conn.ObjectPutBytes(swift.Container, path, contents, ""); err != nil {
		return "", 0, err
	}

	return path, int64(len(contents)), nil
}

func (p SwiftPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	swift, err := getConnInfo(endpoint)
	if err != nil {
		return err
	}

	conn, err := swift.Connect()
	if err != nil {
		return err
	}

	contents, err := conn.ObjectGetBytes(swift.Container, file)
	if err != nil {
		return err
	}

	if _, err = os.Stdout.Write(contents); err != nil {
		return err
	}

	return nil
}

func (p SwiftPlugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	swift, err := getConnInfo(endpoint)
	if err != nil {
		return err
	}

	conn, err := swift.Connect()
	if err != nil {
		return err
	}

	return conn.ObjectDelete(swift.Container, file)
}

func getConnInfo(e plugin.ShieldEndpoint) (info *SwiftConnectionInfo, err error) {
	info = &SwiftConnectionInfo{}
	info.AuthURL, err = e.StringValue("auth_url")
	if err != nil {
		return
	}

	info.ProjectName, err = e.StringValueDefault("project_name", "")
	if err != nil {
		return
	}

	info.Domain, err = e.StringValueDefault("domain", "")
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

func (info SwiftConnectionInfo) Connect() (*swift.Connection, error) {
	conn := &swift.Connection{
		UserName: info.Username,
		ApiKey:   info.Password,
		AuthUrl:  info.AuthURL,
		Domain:   info.Domain,
		Tenant:   info.ProjectName,
	}

	if err := conn.Authenticate(); err != nil {
		return nil, err
	}

	return conn, nil
}
