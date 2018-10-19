package core2

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/jhunt/go-log"
	"gopkg.in/yaml.v2"

	"github.com/starkandwayne/shield/core/bus"
	"github.com/starkandwayne/shield/core/scheduler"
	"github.com/starkandwayne/shield/core/vault"
	"github.com/starkandwayne/shield/db"
)

type Core struct {
	Config Config

	db        *db.DB
	vault     *vault.Client
	providers map[string]AuthProvider
	bus       *bus.Bus
	scheduler *scheduler.Scheduler

	restart bool

	info struct {
		API     int    `json:"api"`
		Version string `json:"version,omitempty"`
		IP      string `json:"ip,omitempty"`
		Env     string `json:"env,omitempty"`
		Color   string `json:"color,omitempty"`
		MOTD    string `json:"motd,omitempty"`
	}
}

type Config struct {
	Debug   bool   `yaml:"debug"`
	DataDir string `yaml:"data-dir"`
	WebRoot string `yaml:"web-root"`

	Scheduler struct {
		FastLoop int `yaml:"fast-loop"`
		SlowLoop int `yaml:"slow-loop"`
		Threads  int `yaml:"threads"`
		Timeout  int `yaml:"timeout"`
	} `yaml:"scheduler"`

	API struct {
		Bind    string `yaml:"bind"`
		Session struct {
			Timeout int `yaml:"timeout"`
		} `yaml:"session"`

		Failsafe struct {
			Username string `yaml:"username"`
			Password string `yaml:"password"`
		} `yaml:"failsafe"`

		Env   string `yaml:"env"`
		Color string `yaml:"color"`
		MOTD  string `yaml:"motd"`
	} `yaml:"api"`

	Auth []struct {
		Name       string `yaml:"name"`
		Identifier string `yaml:"identifier"`
		Backend    string `yaml:"backend"`

		Properties map[interface{}]interface{} `yaml:"properties"`
	} `yaml:"auth"`

	Fabrics []struct {
		Type   string `yaml:"type"`
		SSHKey string `yaml:"ssh-key"`
	} `yaml:"fabrics"`

	Vault struct {
		Address string `yaml:"address"`
		CACert  string `yaml:"ca"`
		ca      string /* PEM-encoded contents */
	} `yaml:"vault"`

	Cipher string `yaml:"cipher"`
}

var (
	Version       string
	DefaultConfig Config
)

func init() {
	DefaultConfig.DataDir = "/shield/data"
	DefaultConfig.WebRoot = "/shield/ui"

	DefaultConfig.Scheduler.FastLoop = 1
	DefaultConfig.Scheduler.SlowLoop = 300
	DefaultConfig.Scheduler.Threads = 5
	DefaultConfig.Scheduler.Timeout = 12 /* hours */

	DefaultConfig.API.Bind = "*:8888"
	DefaultConfig.API.Session.Timeout = 720 /* hours; 30 days */
	DefaultConfig.API.Failsafe.Username = "admin"
	DefaultConfig.API.Failsafe.Password = "shield"

	DefaultConfig.API.Env = "SHIELD"
	DefaultConfig.API.Color = "yellow"

	DefaultConfig.Vault.Address = "http://127.0.0.1:8200"

	DefaultConfig.Cipher = "aes256-ctr"
}

func Configure(file string, config Config) (*Core, error) {
	c := &Core{Config: config}

	if file != "" {
		b, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, err
		}

		if err = yaml.Unmarshal(b, &c.Config); err != nil {
			return nil, err
		}
	}

	/* validate configuration */
	if c.Config.Scheduler.FastLoop <= 0 {
		return nil, fmt.Errorf("scheduler.fast-loop value '%d' is invalid (must be greater than zero)", c.Config.Scheduler.FastLoop)
	}

	if c.Config.Scheduler.SlowLoop <= 0 {
		return nil, fmt.Errorf("scheduler.slow-loop value '%d' is invalid (must be greater than zero)", c.Config.Scheduler.SlowLoop)
	}

	if c.Config.Scheduler.Timeout <= 0 {
		return nil, fmt.Errorf("scheduler.timeout value '%d' hours is invalid (must be greater than zero)", c.Config.Scheduler.Timeout)
	}
	if c.Config.Scheduler.Timeout > 48 {
		return nil, fmt.Errorf("scheduler.timeout value '%d' hours is invalid (must be no larger than 48h)", c.Config.Scheduler.Timeout)
	}

	if c.Config.Scheduler.Threads <= 0 {
		return nil, fmt.Errorf("scheduler.threads value '%d' is invalid (must be greater than zero)", c.Config.Scheduler.Threads)
	}

	if c.Config.API.Session.Timeout <= 0 {
		return nil, fmt.Errorf("api.session.timeout of '%d' hours is invalid (must be greater than zero)", c.Config.API.Session.Timeout)
	}

	if c.Config.Cipher == "" {
		return nil, fmt.Errorf("cipher '%s' is invalid (see documentation for supported ciphers)", c.Config.Cipher)
	}

	if c.Config.DataDir == "" {
		return nil, fmt.Errorf("SHIELD data directory '%s' is invalid (must be a valid path)", c.Config.DataDir)
	}
	if st, err := os.Stat(c.Config.DataDir); err != nil {
		return nil, fmt.Errorf("SHIELD data directory '%s' is invalid (%s)", c.Config.DataDir, err)
	} else if !st.Mode().IsDir() {
		return nil, fmt.Errorf("SHIELD data directory '%s' is invalid (not a directory)", c.Config.DataDir)
	}

	if c.Config.WebRoot == "" {
		return nil, fmt.Errorf("SHIELD web root directory '%s' is invalid (must be a valid path)", c.Config.WebRoot)
	}
	if st, err := os.Stat(c.Config.WebRoot); err != nil {
		return nil, fmt.Errorf("SHIELD web root directory '%s' is invalid (%s)", c.Config.WebRoot, err)
	} else if !st.Mode().IsDir() {
		return nil, fmt.Errorf("SHIELD web root directory '%s' is invalid (not a directory)", c.Config.WebRoot)
	}

	if c.Config.Vault.CACert != "" {
		b, err := ioutil.ReadFile(c.Config.Vault.CACert)
		if err != nil {
			return nil, fmt.Errorf("Unable to read Vault CA Certificate '%s': %s", c.Config.Vault.CACert, err)
		}
		c.Config.Vault.ca = string(b)
	}

	/* set up information for /v2/info and /init.js */
	c.info.API = 2
	c.info.Version = Version
	c.info.IP = ip()
	c.info.MOTD = c.Config.API.MOTD
	c.info.Env = c.Config.API.Env
	c.info.Color = c.Config.API.Color

	/* set up authentication providers */
	c.providers = make(map[string]AuthProvider)
	for i, auth := range c.Config.Auth {
		if auth.Name == "local" {
			return nil, fmt.Errorf("authentication provider #%d is named 'local', which is reserved for internal use by SHIELD itself;please rename this provider", i+1)
		}

		id := auth.Identifier
		if id == "" {
			return nil, fmt.Errorf("provider #%d lacks the required `identifier' field", i+1)
		}
		if auth.Name == "" {
			return nil, fmt.Errorf("%s provider lacks the required `name' field", id)
		}
		if auth.Backend == "" {
			return nil, fmt.Errorf("%s provider lacks the required `backend' field", id)
		}

		switch auth.Backend {
		case "github":
			c.providers[id] = &GithubAuthProvider{
				core: c,
				AuthProviderBase: AuthProviderBase{
					Name:       auth.Name,
					Identifier: id,
					Type:       auth.Backend,
				},
			}
		case "uaa":
			c.providers[id] = &UAAAuthProvider{
				core: c,
				AuthProviderBase: AuthProviderBase{
					Name:       auth.Name,
					Identifier: id,
					Type:       auth.Backend,
				},
			}
		default:
			return nil, fmt.Errorf("%s authentication provider has an unrecognized `backend' of '%s'; must be one of github or uaa", id, auth.Backend)
		}

		if err := c.providers[id].Configure(auth.Properties); err != nil {
			return nil, fmt.Errorf("failed to configure '%s' authentication provider '%s': %s", auth.Backend, id, err)
		}
	}

	return c, nil
}

func (c *Core) Terminate(err error) {
	log.Alertf("SHIELD Core terminating abnormally: %s\n", err)
	os.Exit(3)
}

func (c *Core) MaybeTerminate(err error) {
	if err != nil {
		c.Terminate(err)
	}
}

func (c *Core) Unlocked() bool {
	init, err := c.vault.Initialized()
	if err != nil {
		log.Errorf("unable to check Vault initialization status: %s", err)
		return false
	}
	if init {
		sealed, err := c.vault.Sealed()
		if err != nil {
			log.Errorf("unable to check Vault sealed status: %s", err)
			return false
		}

		return sealed
	}

	return false
}

func (c *Core) DataFile(rel string) string {
	return fmt.Sprintf("%s/%s", c.Config.DataDir, rel)
}

func (c *Core) CryptFile() string {
	return c.DataFile("vault.crypt")
}
