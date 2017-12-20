package s3

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"regexp"
)

const (
	PrivateACL                = "private"
	PublicReadACL             = "public-read"
	PublicReadWriteACL        = "public-read-write"
	AWSExecReadACL            = "aws-exec-read"
	AuthenticatedReadACL      = "authenticated-read"
	BucketOwnerReadACL        = "bucket-owner-read"
	BucketOwnerFullControlACL = "bucket-owner-full-control"
	LogDeliveryWriteACL       = "log-delivery-write"
)

func (c *Client) CreateBucket(name, region, acl string) error {
	/* validate that the bucket name is:

	   - between 3 and 63 characters long (inclusive)
	   - not include periods (for TLS wildcard matching)
	   - lower case
	   - rfc952 compliant
	*/
	if ok, _ := regexp.MatchString(`^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$`, name); !ok {
		return fmt.Errorf("invalid s3 bucket name")
	}

	was := c.Bucket
	defer func() { c.Bucket = was }()
	c.Bucket = name

	b := []byte{}
	if region != "" {
		var payload struct {
			XMLName xml.Name `xml:"CreateBucketConfiguration"`
			Region  string   `xml:"LocationConstraint"`
		}
		payload.Region = region

		var err error
		b, err = xml.Marshal(payload)
		if err != nil {
			return err
		}
	}

	headers := make(http.Header)
	headers.Set("x-amz-acl", acl)

	res, err := c.put("/", b, &headers)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return ResponseError(res)
	}

	return nil
}

func (c *Client) DeleteBucket(name string) error {
	was := c.Bucket
	defer func() { c.Bucket = was }()
	c.Bucket = name

	res, err := c.delete("/", nil)
	if err != nil {
		return err
	}

	if res.StatusCode != 204 {
		return ResponseError(res)
	}

	return nil
}
