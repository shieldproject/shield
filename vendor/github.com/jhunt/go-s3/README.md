go-s3
=====

A simple library for interfacing with Amazon S3 from Go.

Example
-------

```go
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/jhunt/go-s3"
)

func main() {
	/* ... some setup ... */

	c, err := s3.NewClient(&s3.Client{
		AccessKeyID:     aki,
		SecretAccessKey: key,
		Region:          reg,
		Bucket:          bkt,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "!!! unable to configure s3 client: %s\n", err)
		os.Exit(1)
	}

	u, err := c.NewUpload(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "!!! unable to start multipart upload: %s\n", err)
		os.Exit(1)
	}

	n, err := u.Stream(os.Stdin, 5*1024*1024*1024)
	if err != nil {
		fmt.Fprintf(os.Stderr, "!!! unable to stream <stdin> in 5m parts: %s\n", err)
		os.Exit(1)
	}

	err = u.Done()
	if err != nil {
		fmt.Fprintf(os.Stderr, "!!! unable to complete multipart upload: %s\n", err)
		os.Exit(1)
	}
}
```

Environment Variables
---------------------

The following environment variables affect the behavior of this
library:

  - `$S3_TRACE` - If set to "yes", "y", or "1" (case-insensitive),
    any and all HTTP(S) requests will be dumped to standard error.
    If set to the value "header" or "headers", only the headers of
    requests and responses will be dumped.
