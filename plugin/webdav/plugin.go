package main

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	fmt "github.com/jhunt/go-ansi"

	"github.com/starkandwayne/shield/plugin"
)

func main() {
	p := WebDAVPlugin{
		Name:    "WebDAV Plugin",
		Author:  "Stark & Wayne",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
			Target: "no",
			Store:  "yes",
		},
		Example: `
{
  "url"                 : "https://my-blobstore.internal:443/prefix",
  "username"            : "webby",
  "password"            : "sekrit",
  "skip_ssl_validation" : true
}
`,
		Defaults: `
{
}
`,
		Fields: []plugin.Field{
			plugin.Field{
				Mode:     "store",
				Name:     "url",
				Type:     "string",
				Title:    "WebDAV Host",
				Help:     "The URL to the root of the WebDAV host.",
				Required: true,
			},
			plugin.Field{
				Mode:  "store",
				Name:  "username",
				Type:  "string",
				Title: "Username",
				Help:  "Username to authenticate as, via basic auth.",
			},
			plugin.Field{
				Mode:  "store",
				Name:  "password",
				Type:  "password",
				Title: "Password",
				Help:  "Password to authenticate as, via basic auth.",
			},
			plugin.Field{
				Mode:  "store",
				Name:  "skip_ssl_validation",
				Type:  "bool",
				Title: "Skip SSL Validation",
				Help:  "If your WebDAV certificate is invalid, expired, or signed by an unknown Certificate Authority, you can disable SSL validation.  This is not recommended from a security standpoint, however.",
			},
		},
	}

	plugin.Run(p)
}

type WebDAVPlugin plugin.PluginInfo

type WebDAV struct {
	URL        string
	Username   string
	Password   string
	SkipVerify bool

	c *http.Client
}

func (p WebDAVPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p WebDAVPlugin) Validate(endpoint plugin.ShieldEndpoint) error {
	fail := false

	s, err := endpoint.StringValue("url")
	if err != nil {
		fmt.Printf("@R{\u2717 url                  %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 url}                  @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("username", "")
	if err != nil {
		fmt.Printf("@R{\u2717 username             %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 username}             none (no authentication)\n")
	} else {
		fmt.Printf("@G{\u2713 username}             @C{%s} (basic auth)\n", s)
	}

	s, err = endpoint.StringValueDefault("password", "")
	if err != nil {
		fmt.Printf("@R{\u2717 password             %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 password}             none (no authentication)\n")
	} else {
		fmt.Printf("@G{\u2713 password}             @C{%s} (basic auth)\n", s)
	}

	tf, err := endpoint.BooleanValueDefault("skip_ssl_validation", false)
	if err != nil {
		fmt.Printf("@R{\u2717 skip_ssl_validation  %s}\n", err)
		fail = true
	} else if tf {
		fmt.Printf("@G{\u2713 skip_ssl_validation}  @C{yes}, SSL will @Y{NOT} be validated\n")
	} else {
		fmt.Printf("@G{\u2713 skip_ssl_validation}  @C{no}, SSL @Y{WILL} be validated\n")
	}

	if fail {
		return fmt.Errorf("webdav: invalid configuration")
	}
	return nil
}

func (p WebDAVPlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	return plugin.UNIMPLEMENTED
}

func (p WebDAVPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	return plugin.UNIMPLEMENTED
}

func (p WebDAVPlugin) Store(endpoint plugin.ShieldEndpoint) (string, int64, error) {
	dav, err := configure(endpoint)
	if err != nil {
		return "", 0, err
	}

	path := dav.generate()
	plugin.DEBUG("Storing data in %s", path)

	size, err := dav.Put(path, os.Stdin)
	return path, size, err
}

func (p WebDAVPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	dav, err := configure(endpoint)
	if err != nil {
		return err
	}

	return dav.Get(file, os.Stdout)
}

func (p WebDAVPlugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	dav, err := configure(endpoint)
	if err != nil {
		return err
	}

	return dav.Delete(file)
}

func configure(endpoint plugin.ShieldEndpoint) (WebDAV, error) {
	url, err := endpoint.StringValue("url")
	if err != nil {
		return WebDAV{}, err
	}

	username, err := endpoint.StringValueDefault("username", "")
	if err != nil {
		return WebDAV{}, err
	}

	password, err := endpoint.StringValueDefault("password", "")
	if err != nil {
		return WebDAV{}, err
	}

	skip, err := endpoint.BooleanValueDefault("skip_ssl_validation", false)
	if err != nil {
		return WebDAV{}, err
	}

	return WebDAV{
		URL:        url,
		Username:   username,
		Password:   password,
		SkipVerify: skip,
	}, nil
}

func (dav WebDAV) generate() string {
	t := time.Now()
	y, m, d := t.Date()
	H, M, S := t.Clock()
	path := fmt.Sprintf("%04d/%02d/%02d/%04d-%02d-%02d-%02d%02d%02d-%s",
		y, m, d, y, m, d, H, M, S, plugin.GenUUID())
	return strings.Replace(path, "//", "/", -1)
}

func (dav WebDAV) client() *http.Client {
	if dav.c == nil {
		dav.c = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: dav.SkipVerify,
				},
			},
		}
	}
	return dav.c
}

func (dav WebDAV) url(rel string) string {
	u, _ := url.Parse(dav.URL)
	u.Path = fmt.Sprintf("%s/%s", strings.TrimSuffix(u.Path, "/"), rel)
	return u.String()
}

func (dav WebDAV) Put(path string, in io.Reader) (int64, error) {
	req, err := http.NewRequest("PUT", dav.url(path), in)
	if err != nil {
		return 0, err
	}

	res, err := dav.client().Do(req)
	if err != nil {
		return 0, err
	}

	if res.StatusCode == 201 {

		req, err = http.NewRequest("HEAD", dav.url(path), nil)
		if err != nil {
			return 0, err
		}

		res, err = dav.client().Do(req)
		if err != nil {
			return 0, err
		}

		return res.ContentLength, nil
	}

	return 0, fmt.Errorf("Received a %s from %s", res.Status, dav.URL)
}

func (dav WebDAV) Get(path string, out io.Writer) error {
	req, err := http.NewRequest("GET", dav.url(path), nil)
	if err != nil {
		return err
	}

	res, err := dav.client().Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode == 200 {
		_, err := io.Copy(out, res.Body)
		return err
	}

	return fmt.Errorf("Received a %s from %s", res.Status, dav.URL)
}

func (dav WebDAV) Delete(path string) error {
	req, err := http.NewRequest("DELETE", dav.url(path), nil)
	if err != nil {
		return err
	}

	res, err := dav.client().Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode == 200 {
		return nil
	}

	return fmt.Errorf("Received a %s from %s", res.Status, dav.URL)
}
