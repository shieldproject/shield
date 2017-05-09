package api

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"time"
)

var Cfg *Config

type Config struct {
	Backend  string            `yaml:"backend"`
	Backends map[string]string `yaml:"backends"`
	Aliases  map[string]string `yaml:"aliases"`
	Path     string            `yaml:"-"` // omit this little guy

	resolved string // caches the resolved "secure" endpoint in force
}

func LoadConfig(p string) error {
	Cfg = &Config{
		Path:     p,
		Backends: map[string]string{},
		Aliases:  map[string]string{},
	}
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return nil
	}

	data, err := ioutil.ReadFile(p)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, Cfg)
	if err != nil {
		return err
	}

	return nil
}

func (cfg *Config) Save() error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	err = os.MkdirAll(path.Dir(cfg.Path), 0755)
	if err != nil {
		return err
	}

	tempFile, err := ioutil.TempFile(path.Dir(cfg.Path), "shield_config")
	if err != nil {
		return err
	}

	err = os.Chmod(tempFile.Name(), 0600)
	if err != nil {
		return err
	}

	_, err = tempFile.Write(data)
	if err != nil {
		return err
	}

	err = tempFile.Close()
	if err != nil {
		return err
	}

	err = os.Rename(tempFile.Name(), cfg.Path)
	if err != nil {
		return err
	}

	return nil
}

func (cfg *Config) BackendURI() string {
	if cfg.Backend == "" {
		return ""
	}

	return strings.TrimSuffix(cfg.ResolveAlias(cfg.Backend), "/")
}

func (cfg *Config) BackendToken() string {
	return cfg.Backends[cfg.BackendURI()]
}

func (cfg *Config) ResolveAlias(be string) string {
	if _, ok := cfg.Backends[be]; ok {
		return be
	} else if aliasedBE, ok := cfg.Aliases[be]; ok {
		if _, ok := cfg.Backends[aliasedBE]; ok {
			return aliasedBE
		}
	}

	return ""
}

func (cfg *Config) UpdateBackend(be string, token string) error {
	backend := cfg.ResolveAlias(be)
	if backend == "" {
		return fmt.Errorf("Could not find backend '%s' to update", be)
	}
	cfg.Backends[backend] = token
	return nil
}

func (cfg *Config) UpdateCurrentBackend(token string) error {
	return cfg.UpdateBackend(cfg.BackendURI(), token)
}

func (cfg *Config) AddBackend(be string, alias string) error {
	urlRE := regexp.MustCompile(`^https?://[^:]+(:\d+)?/?$`)
	if !urlRE.MatchString(be) {
		return fmt.Errorf("Invalid backend format. Expecting `protocol://hostname:port/`. Got `%s`", be)
	}

	// only reset to "" if it didn't exist previously
	if _, ok := cfg.Backends[be]; !ok {
		cfg.Backends[be] = ""
	}
	cfg.Aliases[alias] = be
	return nil
}

func (cfg *Config) UseBackend(be string) error {
	if cfg.ResolveAlias(be) == "" {
		return fmt.Errorf("Undefined Backend: %s", be)
	}

	cfg.Backend = be
	return nil
}

// Hits the /v1/ping endpoint to trigger any HTTP -> HTTPS redirection
// and then returns the ultimate URL base (minus the '/v1/ping')
func (cfg *Config) SecureBackendURI() (string, error) {
	if cfg.resolved != "" {
		return cfg.resolved, nil
	}

	var final string
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: os.Getenv("SHIELD_SKIP_SSL_VERIFY") != "",
			},
			Proxy:             http.ProxyFromEnvironment,
			DisableKeepAlives: true,
		},
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			final = fmt.Sprintf("%s://%s", req.URL.Scheme, req.URL.Host)
			if len(via) > 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}
	final = Cfg.BackendURI()
	res, err := client.Get(fmt.Sprintf("%s/v1/ping", final))
	if err != nil {
		cfg.resolved = final
		return final, err
	}
	defer res.Body.Close()
	io.Copy(ioutil.Discard, res.Body)
	return final, err
}

func BasicAuthToken(user, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+password))
}
