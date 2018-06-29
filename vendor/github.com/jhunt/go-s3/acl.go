package s3

import (
	"encoding/xml"
	"io/ioutil"
	"net/http"
)

type Grant struct {
	GranteeID   string
	GranteeName string
	Group       string
	Permission  string
}

const EveryoneURI = "http://acs.amazonaws.com/groups/global/AllUsers"

type ACL []Grant

func (c *Client) GetACL(key string) (ACL, error) {
	res, err := c.get(key+"?acl", nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, ResponseError(res)
	}

	var r struct {
		XMLName xml.Name `xml:"AccessControlPolicy"`
		List    struct {
			Grant []struct {
				Grantee struct {
					ID   string `xml:"ID"`
					Name string `xml:"DisplayName"`
					URI  string `xml:"URI"`
				} `xml:"Grantee"`
				Permission string `xml:"Permission"`
			} `xml:"Grant"`
		} `xml:"AccessControlList"`
	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if err := xml.Unmarshal(b, &r); err != nil {
		return nil, err
	}

	var acl ACL
	for _, g := range r.List.Grant {
		group := ""
		if g.Grantee.URI == EveryoneURI {
			group = "EVERYONE"
		}
		acl = append(acl, Grant{
			GranteeID:   g.Grantee.ID,
			GranteeName: g.Grantee.Name,
			Group:       group,
			Permission:  g.Permission,
		})
	}
	return acl, nil
}

func (c *Client) ChangeACL(path, acl string) error {
	headers := make(http.Header)
	headers.Set("x-amz-acl", acl)

	res, err := c.put(path+"?acl", nil, &headers)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return ResponseError(res)
	}

	return nil
}
