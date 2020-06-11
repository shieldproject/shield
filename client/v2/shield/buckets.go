package shield

import (
	"fmt"
	"strings"
)

type Bucket struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Compression string `json:"compression"`
	Encryption  string `json:"encryption"`
}

func (c *Client) ListBuckets() ([]*Bucket, error) {
	var out []*Bucket
	if err := c.get("/v2/buckets", &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetBucket(key string) (*Bucket, error) {
	all, err := c.ListBuckets()
	if err != nil {
		return nil, err
	}

	for _, bucket := range all {
		if bucket.Key == key {
			return bucket, nil
		}
	}

	return nil, fmt.Errorf("bucket not found")
}

func (c *Client) FindBucket(q string, fuzzy bool) (*Bucket, error) {
	l, err := c.FindBuckets(q, fuzzy)
	if err != nil {
		return nil, err
	}

	if len(l) == 0 {
		return nil, fmt.Errorf("no matching bucket found")
	}
	if len(l) > 1 {
		return nil, fmt.Errorf("multiple matching buckets found")
	}

	return l[0], nil
}
func (c *Client) FindBuckets(q string, fuzzy bool) ([]*Bucket, error) {
	all, err := c.ListBuckets()
	if err != nil {
		return nil, err
	}

	matching := make([]*Bucket, 0)
	for _, b := range all {
		if fuzzy && (strings.Contains(b.Key, q) || strings.Contains(b.Name, q)) {
			matching = append(matching, b)
		} else if !fuzzy && (b.Key == q || b.Name == q) {
			matching = append(matching, b)
		}
	}

	return matching, nil
}
