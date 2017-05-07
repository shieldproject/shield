package api

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh/terminal"
)

type URL struct {
	base *url.URL
}

func ParseURL(s string) (*URL, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}

	return &URL{base: u}, nil
}

func (u *URL) AddParameter(key string, value interface{}) error {
	q := u.base.Query()
	switch value.(type) {
	case string:
		q.Add(key, value.(string))
	case bool:
		if value.(bool) {
			q.Add(key, "t")
		} else {
			q.Add(key, "f")
		}
	default:
		q.Add(key, fmt.Sprintf("%v", value))
	}
	u.base.RawQuery = q.Encode()
	return nil
}

func (u *URL) MaybeAddParameter(key string, value interface{}) error {
	if s, ok := value.(string); ok {
		if s != "" {
			return u.AddParameter(key, value)
		}
		return nil
	}

	if yn, ok := value.(YesNo); ok {
		if yn.On {
			return u.AddParameter(key, yn.Yes)
		}
		return nil
	}

	return u.AddParameter(key, value)
}

func (u *URL) String() string {
	return u.base.String()
}

func httpTracingEnabled() bool {
	shieldTrace := os.Getenv("SHIELD_TRACE")
	return shieldTrace != "" && shieldTrace != "0" && shieldTrace != "false" && shieldTrace != "no"
}
func debugRequest(req *http.Request) {
	if httpTracingEnabled() {
		r, _ := httputil.DumpRequest(req, true)
		fmt.Fprintf(os.Stderr, "Request:\n%s\n---------------------------\n", r)
	}
}
func debugResponse(res *http.Response) {
	if httpTracingEnabled() {
		r, _ := httputil.DumpResponse(res, true)
		fmt.Fprintf(os.Stderr, "Response:\n%s\n---------------------------\n", r)
	}
}

func (u *URL) Request(out interface{}, req *http.Request) error {
	var bodyBytes []byte
	var err error
	if req.Body != nil {
		bodyBytes, err = ioutil.ReadAll(req.Body)
		if err != nil {
			return err
		}
		req.Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))
		if err != nil {
			return err
		}
	}

	r, err := makeRequest(req)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	if r.StatusCode == 401 {
		if req.Body != nil {
			req.Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))
		}
		r, err = promptAndAuth(r, req)
		if err != nil {
			return err
		}
	}

	var final error = nil
	if r.StatusCode != 200 {
		final = fmt.Errorf("Error %s", r.Status)
	}

	if out != nil {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return err
		}

		err = json.Unmarshal(body, out)
		if err != nil && final == nil {
			return err
		}
	}

	return final
}

func (u *URL) Get(out interface{}) error {
	r, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return err
	}
	return u.Request(out, r)
}

func (u *URL) Delete(out interface{}) error {
	r, err := http.NewRequest("DELETE", u.String(), nil)
	if err != nil {
		return err
	}
	return u.Request(out, r)
}

func (u *URL) Post(out interface{}, data string) error {
	r, err := http.NewRequest("POST", u.String(),
		bytes.NewBufferString(data))
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/json")
	return u.Request(out, r)
}

func (u *URL) Put(out interface{}, data string) error {
	r, err := http.NewRequest("PUT", u.String(),
		bytes.NewBufferString(data))
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/json")
	return u.Request(out, r)
}

func makeRequest(req *http.Request) (*http.Response, error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: os.Getenv("SHIELD_SKIP_SSL_VERIFY") != "",
			},
			Proxy:             http.ProxyFromEnvironment,
			DisableKeepAlives: true,
		},
		Timeout: 30 * time.Second,
	}
	if os.Getenv("SHIELD_API_TOKEN") != "" {
		req.Header.Set("X-Shield-Token", os.Getenv("SHIELD_API_TOKEN"))
	}
	token := Cfg.BackendToken()
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	debugRequest(req)
	r, err := client.Do(req)
	debugResponse(r)
	if err != nil {
		return nil, err
	}
	return r, err
}

func promptAndAuth(res *http.Response, req *http.Request) (*http.Response, error) {
	auth := strings.Split(res.Header.Get("www-authenticate"), " ")
	if len(auth) > 0 {
		fmt.Fprintf(os.Stdout, "Authentication Required\n\n")
		var token string
		switch strings.ToLower(auth[0]) {
		case "basic":
			var user string
			fmt.Fprintf(os.Stdout, "User: ")
			_, err := fmt.Scanln(&user)
			if err != nil {
				return nil, err
			}
			fmt.Fprintf(os.Stdout, "\nPassword: ")
			pass, err := terminal.ReadPassword(int(os.Stdin.Fd()))
			fmt.Fprintf(os.Stdout, "\n") // newline to line-break after the password prompt
			if err != nil {
				return nil, err
			}

			token = BasicAuthToken(user, string(pass))
		case "bearer":
			var t, s string
			fmt.Fprintf(os.Stdout, "SHIELD has been protected by an OAuth2 provider. To authenticate on the command line,\n please visit the following URL, and paste in the entirety of the Bearer token provided:\n\n\t%s/v1/auth/cli\n\nToken: ", Cfg.BackendURI())
			_, err := fmt.Scanln(&t, &s)
			if err != nil {
				return nil, err
			}
			token = t + " " + s
		default:
			return nil, fmt.Errorf("Unrecognized authentication request type: %s", res.Header.Get("www-authenticate"))
		}
		// newline to separate creds from response
		fmt.Fprintf(os.Stdout, "\n")

		Cfg.UpdateCurrentBackend(token)
		r, err := makeRequest(req)
		if err != nil {
			return nil, err
		}
		if r.StatusCode < 400 {
			Cfg.Save()
		}
		return r, nil
	}
	// if no authorization header, fall back to returning the orignal response, unprocessed
	return res, nil
}
