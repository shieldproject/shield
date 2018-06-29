package s3

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"
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

type Bucket struct {
	Name         string
	CreationDate time.Time
	OwnerID      string
	OwnerName    string
}

func (c *Client) ListBuckets() ([]Bucket, error) {
	prev := c.Bucket
	c.Bucket = ""
	res, err := c.get("/", nil)
	c.Bucket = prev
	if err != nil {
		return nil, err
	}

	var r struct {
		XMLName xml.Name `xml:"ListAllMyBucketsResult"`
		Owner   struct {
			ID          string `xml:"ID"`
			DisplayName string `xml:"DisplayName"`
		} `xml:"Owner"`
		Buckets struct {
			Bucket []struct {
				Name         string `xml:"Name"`
				CreationDate string `xml:"CreationDate"`
			} `xml:"Bucket"`
		} `xml:"Buckets"`
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, ResponseErrorFrom(b)
	}

	err = xml.Unmarshal(b, &r)
	if err != nil {
		return nil, err
	}

	s := make([]Bucket, len(r.Buckets.Bucket))
	for i, bkt := range r.Buckets.Bucket {
		s[i].OwnerID = r.Owner.ID
		s[i].OwnerName = r.Owner.DisplayName
		s[i].Name = bkt.Name

		created, _ := time.Parse("2006-01-02T15:04:05.000Z", bkt.CreationDate)
		s[i].CreationDate = created
	}
	return s, nil
}
