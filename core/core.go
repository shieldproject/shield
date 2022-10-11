package core

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	env "github.com/jhunt/go-envirotron"
	"github.com/jhunt/go-log"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"

	"github.com/shieldproject/shield/core/bus"
	"github.com/shieldproject/shield/core/fabric"
	"github.com/shieldproject/shield/core/metrics"
	"github.com/shieldproject/shield/core/scheduler"
	"github.com/shieldproject/shield/core/vault"
	"github.com/shieldproject/shield/db"
)

type Core struct {
	Config Config

	db        *db.DB
	vault     *vault.Client
	providers map[string]AuthProvider
	bus       *bus.Bus
	scheduler *scheduler.Scheduler
	metrics   *metrics.Exporter

	bailout bool

	info struct {
		API     int    `json:"api"`
		Version string `json:"version,omitempty"`
		Env     string `json:"env,omitempty"`
		Color   string `json:"color,omitempty"`
		MOTD    string `json:"motd,omitempty"`
	}
}

type Config struct {
	Debug          bool     `yaml:"debug"          env:"SHIELD_DEBUG"`
	DataDir        string   `yaml:"data-dir"       env:"SHIELD_DATA_DIR"`
	WebRoot        string   `yaml:"web-root"       env:"SHIELD_WEB_ROOT"`
	PluginPaths    []string `yaml:"plugin_paths"`
	PluginPathsEnv string   `yaml:"-"              env:"SHIELD_PLUGIN_PATHS"`

	Scheduler struct {
		FastLoop duration `yaml:"fast-loop" env:"SHIELD_SCHEDULER_FAST_LOOP"`
		SlowLoop duration `yaml:"slow-loop" env:"SHIELD_SCHEDULER_SLOW_LOOP"`
		Threads  int      `yaml:"threads"   env:"SHIELD_SCHEDULER_THREADS"`
		Timeout  int      `yaml:"timeout"   env:"SHIELD_SCHEDULER_TIMEOUT"`
	} `yaml:"scheduler"`

	API struct {
		Bind    string `yaml:"bind"  env:"SHIELD_API_BIND"`
		PProf   string `yaml:"pprof" env:"SHIELD_API_PPROF"`
		Session struct {
			ClearOnBoot bool     `yaml:"clear-on-boot" env:"SHIELD_API_SESSION_CLEAR_ON_BOOT"`
			Timeout     duration `yaml:"timeout"       env:"SHIELD_API_SESSION_TIMEOUT"`
		} `yaml:"session"`

		Failsafe struct {
			Username string `yaml:"username" env:"SHIELD_API_FAILSAFE_USERNAME"`
			Password string `yaml:"password" env:"SHIELD_API_FAILSAFE_PASSWORD"`
		} `yaml:"failsafe"`

		Websocket struct {
			WriteTimeout duration `yaml:"write-timeout" env:"SHIELD_API_WEBSOCKET_WRITE_TIMEOUT"`
			PingInterval duration `yaml:"ping-interval" env:"SHIELD_API_WEBSOCKET_PING_INTERVAL"`
		} `yaml:"websocket"`

		Env   string `yaml:"env"   env:"SHIELD_API_ENV"`
		Color string `yaml:"color" env:"SHIELD_API_COLOR"`
		MOTD  string `yaml:"motd"  env:"SHIELD_API_MOTD"`
	} `yaml:"api"`

	Limit struct {
		Retention struct {
			Min int `yaml:"min" env:"SHIELD_LIMIT_RETENTION_MIN"`
			Max int `yaml:"max" env:"SHIELD_LIMIT_RETENTION_MAX"`
		} `yaml:"retention"`
	} `yaml:"limit"`

	Metadata struct {
		Retention struct {
			PurgedArchives duration `yaml:"purged_archives" env:"SHIELD_METADATA_RETENTION_PURGED_ARCHIVES"`
			TaskLogs       duration `yaml:"task_logs"       env:"SHIELD_METADATA_RETENTION_TASK_LOGS"`
		} `yaml:"retention"`
	} `yaml:"metadata"`

	Auth []struct {
		Name       string `yaml:"name"`
		Identifier string `yaml:"identifier"`
		Backend    string `yaml:"backend"`

		Properties map[interface{}]interface{} `yaml:"properties"`
	} `yaml:"auth"`

	LegacyAgents struct {
		Enabled     bool     `yaml:"enabled"      env:"SHIELD_LEGACY_AGENTS_ENABLED"`
		PrivateKey  string   `yaml:"private-key"  env:"SHIELD_LEGACY_AGENTS_PRIVATE_KEY"`
		DialTimeout duration `yaml:"dial-timeout" env:"SHIELD_LEGACY_AGENTS_DIAL_TIMEOUT"`
		MACs        []string `yaml:"macs"`

		cc  *ssh.ClientConfig
		pub string
	} `yaml:"legacy-agents"`

	Vault struct {
		Address string `yaml:"address" env:"SHIELD_VAULT_ADDRESS"`
		CACert  string `yaml:"ca"      env:"SHIELD_VAULT_CA"`
	} `yaml:"vault"`

	Mbus struct {
		MaxSlots int `yaml:"max-slots" env:"SHIELD_MBUS_MAX_SLOTS"`
		Backlog  int `yaml:"backlog"   env:"SHIELD_MBUS_BACKLOG"`
	} `yaml:"mbus"`

	Prometheus struct {
		Namespace string `yaml:"namespace" env:"SHIELD_PROMETHEUS_NAMESPACE"`

		Username string `yaml:"username"   env:"SHIELD_PROMETHEUS_USERNAME"`
		Password string `yaml:"password"   env:"SHIELD_PROMETHEUS_PASSWORD"`
		Realm    string `yaml:"realm"      env:"SHIELD_PROMETHEUS_REALM"`
	} `yaml:"prometheus"`

	Cipher string `yaml:"cipher" env:"SHIELD_CIPHER"`

	Bootstrapper string `yaml:"bootstrapper" env:"SHIELD_BOOTSTRAPPER"`
}

var (
	Version       string
	DefaultConfig Config
)

func init() {
	DefaultConfig.DataDir = "/shield/data"
	DefaultConfig.WebRoot = "/shield/ui"
	DefaultConfig.PluginPathsEnv = "/shield/plugins"

	DefaultConfig.Scheduler.FastLoop = 1
	DefaultConfig.Scheduler.SlowLoop = 300
	DefaultConfig.Scheduler.Threads = 5
	DefaultConfig.Scheduler.Timeout = 12 /* hours */

	DefaultConfig.API.Bind = "*:8888"
	DefaultConfig.API.Session.Timeout = 720 /* hours; 30 days */
	DefaultConfig.API.Failsafe.Username = "admin"
	DefaultConfig.API.Failsafe.Password = "password"

	DefaultConfig.API.Env = "SHIELD"
	DefaultConfig.API.Color = "yellow"

	DefaultConfig.API.Websocket.WriteTimeout = 45
	DefaultConfig.API.Websocket.PingInterval = 30

	DefaultConfig.Limit.Retention.Min = 1
	DefaultConfig.Limit.Retention.Max = 390

	DefaultConfig.Metadata.Retention.PurgedArchives = 60 * 60 * 24 * 90
	DefaultConfig.Metadata.Retention.TaskLogs = 60 * 60 * 24 * 90

	DefaultConfig.LegacyAgents.Enabled = true
	DefaultConfig.LegacyAgents.DialTimeout = 30

	DefaultConfig.Vault.Address = "http://127.0.0.1:8200"

	DefaultConfig.Cipher = "aes256-ctr"

	DefaultConfig.Bootstrapper = "shield-restarter"

	DefaultConfig.Mbus.MaxSlots = 2048
	DefaultConfig.Mbus.Backlog = 100

	DefaultConfig.Prometheus.Namespace = "shield"
	DefaultConfig.Prometheus.Username = "prometheus"
	DefaultConfig.Prometheus.Password = "shield"
	DefaultConfig.Prometheus.Realm = "SHIELD Prometheus Exporter"
}

func Configure(file string, config Config) (*Core, error) {
	c := &Core{Config: config}
	env.Override(&c.Config)

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

	if c.Config.API.Websocket.PingInterval <= 0 {
		return nil, fmt.Errorf("api.websocket.ping-interval of '%d' seconds is invalid (must be greater than zero)", c.Config.API.Websocket.PingInterval)
	}

	if c.Config.API.Websocket.WriteTimeout <= 0 {
		return nil, fmt.Errorf("api.websocket.write-timeout of '%d' seconds is invalid (must be greater than zero)", c.Config.API.Websocket.WriteTimeout)
	}

	if c.Config.Mbus.MaxSlots <= 0 {
		return nil, fmt.Errorf("mbus.max-slots of '%d' is invalid (must be greater than zero)", c.Config.Mbus.MaxSlots)
	}

	if c.Config.Mbus.Backlog < 0 {
		return nil, fmt.Errorf("mbus.backlog of '%d' is invalid (must be a positive integer)", c.Config.Mbus.Backlog)
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

	if len(c.Config.PluginPaths) == 0 && c.Config.PluginPathsEnv != "" {
		p := strings.Split(c.Config.PluginPathsEnv, ":")
		for _, path := range c.Config.PluginPaths {
			p = append(p, path)
		}
		c.Config.PluginPaths = p
	}
	for _, path := range c.Config.PluginPaths {
		if path == "" {
			return nil, fmt.Errorf("SHIELD plugin directory '%s' is invalid (must be a valid path)", path)
		}
		if st, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("SHIELD plugin directory '%s' is invalid (%s)", path, err)
		} else if !st.Mode().IsDir() {
			return nil, fmt.Errorf("SHIELD plugin directory '%s' is invalid (not a directory)", path)
		}
	}

	if c.Config.Vault.CACert != "" {
		if !strings.HasPrefix(c.Config.Vault.CACert, "---") {
			b, err := ioutil.ReadFile(c.Config.Vault.CACert)
			if err != nil {
				return nil, fmt.Errorf("Unable to read Vault CA Certificate '%s': %s", c.Config.Vault.CACert, err)
			}
			c.Config.Vault.CACert = string(b)
		}
	}

	if !c.Config.LegacyAgents.Enabled {
		return nil, fmt.Errorf("Agent communication has been disabled.  Please set legacy-agents.enabled to 'yes'")
	}
	if c.Config.LegacyAgents.Enabled {
		if c.Config.LegacyAgents.PrivateKey == "" {
			return nil, fmt.Errorf("No SSH private key provided for communicating with legacy agents")
		}
		signer, err := ssh.ParsePrivateKey([]byte(c.Config.LegacyAgents.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("Invalid SSH private key provided for communicating with legacy agents: %s", err)
		}
		c.Config.LegacyAgents.pub = fmt.Sprintf("%s %s",
			signer.PublicKey().Type(),
			base64.StdEncoding.EncodeToString(signer.PublicKey().Marshal()))

		if c.Config.LegacyAgents.DialTimeout < 0 {
			return nil, fmt.Errorf("Invalid connection timeout provided for communicating with legacy agents: %d is less than 0", c.Config.LegacyAgents.DialTimeout)
		}

		if len(c.Config.LegacyAgents.MACs) == 0 {
			c.Config.LegacyAgents.MACs = []string{"hmac-sha2-256-etm@openssh.com", "hmac-sha2-256", "hmac-sha1"}
		}
		c.Config.LegacyAgents.cc = &ssh.ClientConfig{
			Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         time.Duration(c.Config.LegacyAgents.DialTimeout) * time.Second,
			Config:          ssh.Config{MACs: c.Config.LegacyAgents.MACs},
		}
	}

	/* set up information for /v2/info and /init.js */
	c.info.API = 2
	c.info.Version = Version
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
				/* we will provide a link back to core in c.WireUpAuthenticationProviders() */
				AuthProviderBase: AuthProviderBase{
					Name:       auth.Name,
					Identifier: id,
					Type:       auth.Backend,
				},
			}
		case "uaa":
			c.providers[id] = &UAAAuthProvider{
				/* we will provide a link back to core in c.WireUpAuthenticationProviders() */
				AuthProviderBase: AuthProviderBase{
					Name:       auth.Name,
					Identifier: id,
					Type:       auth.Backend,
				},
			}
		case "okta":
			c.providers[id] = &OktaAuthProvider{
				/* we will provide a link back to core in c.WireUpAuthenticationProviders() */
				AuthProviderBase: AuthProviderBase{
					Name:       auth.Name,
					Identifier: id,
					Type:       auth.Backend,
				},
			}
		default:
			return nil, fmt.Errorf("%s authentication provider has an unrecognized `backend' of '%s'; must be one of github, uaa or okta", id, auth.Backend)
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
	status, err := c.vault.Status()
	if err != nil {
		log.Errorf("unable to check Vault status: %s", err)
		return false
	}
	return status == vault.Ready
}

func (c *Core) DataFile(rel string) string {
	return fmt.Sprintf("%s/%s", c.Config.DataDir, rel)
}

func (c *Core) CryptFile() string {
	return c.DataFile("vault.crypt")
}

func (c *Core) FabricFor(task *db.Task) (fabric.Fabric, error) {
	return fabric.Legacy(task.Agent, c.Config.LegacyAgents.cc, c.db), nil
}
