package s3

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"time"
)

func (c *Client) List() ([]Object, error) {
	objects := make([]Object, 0)
	ctok := ""
	for {
		res, err := c.get(fmt.Sprintf("/?list-type=2%s", ctok), nil)
		if err != nil {
			return nil, err
		}

		var r struct {
			XMLName  xml.Name `xml:"ListBucketResult"`
			Next     string   `xml:"NextContinuationToken"`
			Contents []struct {
				Key          string `xml:"Key"`
				LastModified string `xml:"LastModified"`
				ETag         string `xml:"ETag"`
				Size         int64  `xml:"Size"`
				StorageClass string `xml:"StorageClass"`
				Owner        struct {
					ID          string `xml:"ID"`
					DisplayName string `xml:"DisplayName"`
				} `xml:"Owner"`
			} `xml:"Contents"`
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

		for _, f := range r.Contents {
			mod, _ := time.Parse("2006-01-02T15:04:05.000Z", f.LastModified)
			objects = append(objects, Object{
				Key:          f.Key,
				LastModified: mod,
				ETag:         f.ETag[1 : len(f.ETag)-1],
				Size:         Bytes(f.Size),
				StorageClass: f.StorageClass,
				OwnerID:      f.Owner.ID,
				OwnerName:    f.Owner.DisplayName,
			})
		}

		if r.Next == "" {
			return objects, nil
		}

		ctok = fmt.Sprintf("&continuation-token=%s", r.Next)
	}
}
