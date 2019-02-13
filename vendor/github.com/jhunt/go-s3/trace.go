package s3

import (
	"io"
	"net/http"
	"net/http/httputil"

	fmt "github.com/jhunt/go-ansi"
)

func (c *Client) Trace(out io.Writer, yes, body bool) {
	c.traceTo = out
	c.trace = yes || body
	c.traceBody = body
}

func (c *Client) traceRequest(r *http.Request) error {
	if c.trace {
		what, err := httputil.DumpRequest(r, c.traceBody)
		if err != nil {
			return err
		}
		fmt.Fprintf(c.traceTo, "---[ request ]-----------------------------------\n@C{%s}\n\n", what)
	}
	return nil
}

func (c *Client) traceResponse(r *http.Response) error {
	if c.trace {
		what, err := httputil.DumpResponse(r, c.traceBody)
		if err != nil {
			return err
		}
		fmt.Fprintf(c.traceTo, "---[ response ]----------------------------------\n@W{%s}\n\n", what)
	}
	return nil
}
