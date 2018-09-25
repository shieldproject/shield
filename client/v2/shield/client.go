package shield

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

type Client struct {
	URL   string
	Debug bool
	Trace bool

	InsecureSkipVerify bool
	TrustSystemCAs     bool
	CACertificate      string

	Timeout int
	Session string

	ua   *http.Client
	init bool
}

func (c *Client) initialize() error {
	if c.init {
		return nil
	}

	/* drop trailing slashes */
	for strings.HasSuffix(c.URL, "/") {
		c.URL = strings.TrimSuffix(c.URL, "/")
	}

	/* set a default timeout */
	if c.Timeout == 0 {
		c.Timeout = 45
	}

	/* initialize a user agent */
	if c.ua == nil {
		/* set up a certificate pool */
		var pool *x509.CertPool

		/* do we trust the system ca certificates? */
		if c.TrustSystemCAs {
			pool, _ = x509.SystemCertPool()
		}
		if pool == nil {
			pool = x509.NewCertPool()
		}

		/* add the explicit ca certificate */
		if c.CACertificate != "" && !pool.AppendCertsFromPEM([]byte(c.CACertificate)) {
			return fmt.Errorf("Unable to parse CA Certificate for inclusion in trusted CA pool")
		}

		/* set up the client we will use on all requests */
		c.ua = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: c.InsecureSkipVerify,
					RootCAs:            pool,
				},
				Proxy:             http.ProxyFromEnvironment,
				DisableKeepAlives: true,
			},
			Timeout: time.Duration(c.Timeout) * time.Second,
		}
	}

	c.init = true
	return nil
}

func (c *Client) curl(req *http.Request) (*http.Response, error) {
	err := c.initialize()
	if err != nil {
		return nil, err
	}

	if c.Session != "" {
		req.Header.Set("X-Shield-Session", c.Session)
	}

	if req.URL.Scheme == "" {
		req.URL, err = url.Parse(c.URL + req.URL.String())
		if err != nil {
			return nil, err
		}
	}

	if c.Trace {
		r, _ := httputil.DumpRequest(req, true)
		fmt.Fprintf(os.Stderr, "Request:\n%s\n---------------------------\n", r)
	}

	res, err := c.ua.Do(req)
	if err != nil {
		return nil, err
	}

	if c.Trace {
		r, _ := httputil.DumpResponse(res, true)
		fmt.Fprintf(os.Stderr, "Response:\n%s\n---------------------------\n", r)
	}

	return res, nil
}

func (c *Client) request(req *http.Request, out interface{}) error {
	res, err := c.curl(req)
	if err != nil {
		return err
	}

	if res.StatusCode == 204 {
		return nil
	}

	if res.StatusCode == 200 {
		if session := res.Header.Get("X-Shield-Session"); session != "" {
			c.Session = session
		}

		if out == nil {
			return nil
		}

		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		return json.Unmarshal(b, out)
	}

	var e Error
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &e); err != nil {
		return err
	}
	return e
}

func (c *Client) get(path string, out interface{}) error {
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	return c.request(req, out)
}

func (c *Client) post(path string, in, out interface{}) error {
	b, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", path, bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-type", "application/json")
	return c.request(req, out)
}

func (c *Client) put(path string, in, out interface{}) error {
	b, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", path, bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-type", "application/json")
	return c.request(req, out)
}

func (c *Client) patch(path string, in, out interface{}) error {
	b, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PATCH", path, bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-type", "application/json")
	return c.request(req, out)
}

func (c *Client) delete(path string, out interface{}) error {
	req, err := http.NewRequest("DELETE", path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	return c.request(req, out)
}
