package supervisor

import (
	"fmt"
	"net/http"
	"strings"

	"encoding/gob"
	"github.com/antonlindstrom/pgstore"
	"github.com/gorilla/securecookie"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/cloudfoundry"
	"github.com/markbates/goth/providers/github"
	"github.com/michaeljs1990/sqlitestore"
	"github.com/pborman/uuid"
	"github.com/starkandwayne/goutils/log"
	"github.com/starkandwayne/shield/db"
)

type WebServer struct {
	Database      *db.DB
	Addr          string
	WebRoot       string
	Auth          AuthConfig
	Authenticator http.Handler
	Supervisor    *Supervisor
}

func (ws *WebServer) Setup() error {
	var err error
	log.Debugf("Configuring WebServer...")
	if err := ws.Database.Connect(); err != nil {
		log.Errorf("Failed to connect to %s database at %s: %s", ws.Database.Driver, ws.Database.DSN, err)
		return err
	}

	if ws.Auth.OAuth.Provider != "" {
		log.Debugf("Configuring OAuth Session store")
		maxSessionAge := ws.Auth.OAuth.Sessions.MaxAge
		authKey := securecookie.GenerateRandomKey(64)
		encKey := securecookie.GenerateRandomKey(32)
		switch ws.Auth.OAuth.Sessions.Type {
		case "sqlite3":
			log.Debugf("Using sqlite3 as a session store")
			store, err := sqlitestore.NewSqliteStore(ws.Auth.OAuth.Sessions.DSN, "http_sessions", "/", maxSessionAge, authKey, encKey)
			if err != nil {
				log.Errorf("Error setting up sessions database: %s", err)
				return err
			}
			gothic.Store = store
		case "postgres":
			log.Debugf("Using postgres as a session store")
			gothic.Store = pgstore.NewPGStore(ws.Auth.OAuth.Sessions.DSN, authKey, encKey)
			gothic.Store.(*pgstore.PGStore).Options.MaxAge = maxSessionAge
		case "mock":
			log.Debugf("Using mocked session store")
			// does nothing, to avoid being accidentally used in prod
		default:
			log.Errorf("Invalid DB Backend for OAuth sessions database")
			return err
		}

		gob.Register(map[string]interface{}{})
		switch ws.Auth.OAuth.Provider {
		case "github":
			log.Debugf("Using github as the oauth provider")
			goth.UseProviders(github.New(ws.Auth.OAuth.Key, ws.Auth.OAuth.Secret, fmt.Sprintf("%s/v1/auth/github/callback", ws.Auth.OAuth.BaseURL), "read:org"))
			OAuthVerifier = &GithubVerifier{Orgs: ws.Auth.OAuth.Authorization.Orgs}
		case "cloudfoundry":
			log.Debugf("Using cloudfoundry as the oauth provider")
			goth.UseProviders(cloudfoundry.New(ws.Auth.OAuth.ProviderURL, ws.Auth.OAuth.Key, ws.Auth.OAuth.Secret, fmt.Sprintf("%s/v1/auth/cloudfoundry/callback", ws.Auth.OAuth.BaseURL), "openid,scim.read"))
			OAuthVerifier = &UAAVerifier{Groups: ws.Auth.OAuth.Authorization.Orgs, UAA: ws.Auth.OAuth.ProviderURL}
			p, err := goth.GetProvider("cloudfoundry")
			if err != nil {
				return err
			}
			p.(*cloudfoundry.Provider).Client = ws.Auth.OAuth.Client
		case "faux":
			log.Debugf("Using mocked session store")
			// does nothing, to avoid being accidentally used in prod
		default:
			log.Errorf("Invalid OAuth provider specified.")
			return err
		}

		gothic.GetProviderName = func(req *http.Request) (string, error) {
			return ws.Auth.OAuth.Provider, nil
		}

		gothic.SetState = func(req *http.Request) string {
			sess, _ := gothic.Store.Get(req, gothic.SessionName)
			sess.Values["state"] = uuid.New()
			return sess.Values["state"].(string)
		}
	}

	protectedAPIs, err := ws.ProtectedAPIs()
	if err != nil {
		log.Errorf("Could not set up HTTP routes: " + err.Error())
		return err
	}

	if ws.Auth.OAuth.Provider != "" {
		log.Debugf("Enabling OAuth handlers for HTTP requests")
		UserAuthenticator = OAuthenticator{
			Cfg: ws.Auth.OAuth,
		}
	} else {
		log.Debugf("Enabling Basic Auth handlers for HTTP requests")
		UserAuthenticator = BasicAuthenticator{
			Cfg: ws.Auth.Basic,
		}
	}

	http.Handle("/", ws.UnauthenticatedResources(Authenticate(ws.Auth.Tokens, protectedAPIs)))
	return nil
}

func (ws *WebServer) Start() {
	err := ws.Setup()
	if err != nil {
		panic("Could not set up WebServer for SHIELD: " + err.Error())
	}
	log.Debugf("Starting WebServer on '%s'...", ws.Addr)
	err = http.ListenAndServe(ws.Addr, nil)
	if err != nil {
		log.Errorf("HTTP API failed %s", err.Error())
		panic("Cannot setup WebServer, aborting. Check logs for details.")
	}
}

func (ws *WebServer) UnauthenticatedResources(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ws.Auth.OAuth.Provider != "" {
			if r.URL.Path == "/v1/auth/"+ws.Auth.OAuth.Provider+"/callback" {
				UserAuthenticator.(OAuthenticator).OAuthCallback().ServeHTTP(w, r)
				return
			}
		}

		if r.URL.Path == "/v1/ping" {
			ping := &PingAPI{}
			ping.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/v1/meta/") {
			meta := &MetaAPI{PrivateKeyFile: ws.Supervisor.PrivateKeyFile}
			meta.ServeHTTP(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (ws *WebServer) ProtectedAPIs() (http.Handler, error) {
	router := http.NewServeMux()

	status := &StatusAPI{
		Data:  ws.Database,
		Super: ws.Supervisor,
	}
	router.Handle("/v1/status", status)
	router.Handle("/v1/status/", status)

	jobs := &JobAPI{
		Data:       ws.Database,
		ResyncChan: ws.Supervisor.resync,
		Tasks:      ws.Supervisor.adhoc,
	}
	router.Handle("/v1/jobs", jobs)
	router.Handle("/v1/jobs/", jobs)
	router.Handle("/v1/job", jobs)
	router.Handle("/v1/job/", jobs)

	retention := &RetentionAPI{
		Data:       ws.Database,
		ResyncChan: ws.Supervisor.resync,
	}
	router.Handle("/v1/retention", retention)
	router.Handle("/v1/retention/", retention)

	archives := &ArchiveAPI{
		Data:       ws.Database,
		ResyncChan: ws.Supervisor.resync,
		Tasks:      ws.Supervisor.adhoc,
	}
	router.Handle("/v1/archives", archives)
	router.Handle("/v1/archives/", archives)
	router.Handle("/v1/archive", archives)
	router.Handle("/v1/archive/", archives)

	schedules := &ScheduleAPI{
		Data:       ws.Database,
		ResyncChan: ws.Supervisor.resync,
	}
	router.Handle("/v1/schedules", schedules)
	router.Handle("/v1/schedules/", schedules)
	router.Handle("/v1/schedule", schedules)
	router.Handle("/v1/schedule/", schedules)

	stores := &StoreAPI{
		Data:       ws.Database,
		ResyncChan: ws.Supervisor.resync,
	}
	router.Handle("/v1/stores", stores)
	router.Handle("/v1/stores/", stores)
	router.Handle("/v1/store", stores)
	router.Handle("/v1/store/", stores)

	targets := &TargetAPI{
		Data:       ws.Database,
		ResyncChan: ws.Supervisor.resync,
	}
	router.Handle("/v1/targets", targets)
	router.Handle("/v1/targets/", targets)
	router.Handle("/v1/target", targets)
	router.Handle("/v1/target/", targets)

	tasks := &TaskAPI{
		Data: ws.Database,
	}
	router.Handle("/v1/tasks", tasks)
	router.Handle("/v1/tasks/", tasks)
	router.Handle("/v1/task", tasks)
	router.Handle("/v1/task/", tasks)

	jwtCreator := &JWTCreator{
		SigningKey: ws.Auth.OAuth.JWTPrivateKey,
	}
	router.Handle("/v1/auth/cli", jwtCreator)

	router.Handle("/", http.FileServer(http.Dir(ws.WebRoot)))
	return router, nil
}
