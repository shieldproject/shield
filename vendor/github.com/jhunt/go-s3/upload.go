package s3

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
)

type xmlpart struct {
	PartNumber int    `xml:"PartNumber"`
	ETag       string `xml:"ETag"`
}

type Upload struct {
	Key string

	c    *Client
	n    int
	id   string
	sig  string
	path string

	parts []xmlpart
}

func (c *Client) NewUpload(path string) (*Upload, error) {
	res, err := c.post(path+"?uploads", nil, nil)
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, ResponseErrorFrom(b)
	}

	var payload struct {
		Bucket   string `xml:"Bucket"`
		Key      string `xml:"Key"`
		UploadId string `xml:"UploadId"`
	}
	err = xml.Unmarshal(b, &payload)
	if err != nil {
		return nil, err
	}

	return &Upload{
		Key: payload.Key,

		c:    c,
		id:   payload.UploadId,
		path: path,
		n:    0,
	}, nil
}

func (u *Upload) Write(b []byte) error {
	if u.n == 0 {
		u.n = 1
	}
	if u.n > 10000 {
		return fmt.Errorf("S3 limits the number of multipart upload segments to 10k")
	}

	res, err := u.c.put(fmt.Sprintf("%s?partNumber=%d&uploadId=%s", u.path, u.n, u.id), b, nil)
	if err != nil {
		return err
	}

	u.parts = append(u.parts, xmlpart{
		PartNumber: u.n,
		ETag:       res.Header.Get("ETag"),
	})
	u.n++
	return nil
}

func (u *Upload) Done() error {
	var payload struct {
		XMLName xml.Name  `xml:"CompleteMultipartUpload"`
		Parts   []xmlpart `xml:"Part"`
	}
	payload.Parts = u.parts

	b, err := xml.Marshal(payload)
	if err != nil {
		return err
	}

	res, err := u.c.post(fmt.Sprintf("%s?uploadId=%s", u.path, u.id), b, nil)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return ResponseError(res)
	}
	return nil
}

func (u *Upload) Stream(in io.Reader, block int) (int64, error) {
	if block < 5*1024*1024 {
		return 0, fmt.Errorf("S3 requires block sizes of 5MB or higher")
	}

	var total int64
	buf := make([]byte, block)
	for {
		nread, err := io.ReadAtLeast(in, buf, block)
		if err != nil && err != io.ErrUnexpectedEOF {
			if err == io.EOF {
				return total, nil
			}
			return total, err
		}

		werr := u.Write(buf[0:nread])
		if werr != nil {
			return total, werr
		}
		total += int64(nread)

		if err == io.ErrUnexpectedEOF {
			return total, nil
		}
	}
}
