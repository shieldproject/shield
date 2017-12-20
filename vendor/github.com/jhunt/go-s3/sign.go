package s3

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

func v2Resource(bucket string, req *http.Request) []byte {
	r := []byte(fmt.Sprintf("/%s%s", bucket, req.URL.Path))
	if req.URL.RawQuery == "" {
		return r
	}

	qq := strings.Split(req.URL.RawQuery, "&")
	sort.Strings(qq)

	ll := make([][]byte, len(qq))
	for i := range qq {
		kv := strings.SplitN(qq[i], "=", 2)
		k := uriencode(kv[0], true)
		if len(kv) == 2 {
			v := uriencode(kv[1], true)

			ll[i] = make([]byte, len(k)+1+len(v))
			copy(ll[i], k)
			ll[i][len(k)] = 0x3d
			copy(ll[i][1+len(k):], v)

		} else {
			ll[i] = k
		}
	}

	return bytes.Join([][]byte{
		r,
		bytes.Join(ll, []byte{0x26}),
	}, []byte{0x3f})
}

func mac256(key, msg []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(msg)
	return h.Sum(nil)
}

func (c *Client) signature(req *http.Request, raw []byte) string {
	if c.SignatureVersion == 2 {
		return c.v2signature(req, raw)
	}
	if c.SignatureVersion == 4 {
		return c.v4signature(req, raw)
	}
	panic(fmt.Sprintf("unrecognized aws/s3 signature version %d", c.SignatureVersion))
}

func v2Headers(req *http.Request) []byte {
	subset := make(map[string]string)
	names := make([]string, 0)

	for header := range req.Header {
		lc := strings.ToLower(header)
		if strings.HasPrefix(lc, "x-amz-") {
			names = append(names, lc)
			subset[lc] = strings.Trim(req.Header.Get(header), " \t\r\n\f") + "\n"
		}
	}
	sort.Strings(names)

	ll := make([][]byte, len(names))
	for i, header := range names {
		ll[i] = bytes.Join([][]byte{[]byte(header), []byte(subset[header])}, []byte{0x3a})
	}
	return bytes.Join(ll, []byte{})
}

func (c *Client) v2signature(req *http.Request, raw []byte) string {
	now := time.Now().UTC()

	req.Header.Set("x-amz-date", now.Format("20060102T150405Z"))
	req.Header.Set("host", regexp.MustCompile(`:.*`).ReplaceAllString(req.URL.Host, ""))
	//req.Header.Set("host", "go-s3-bd6cf051-8023-4d2b-8bf2-7aaa477862ea.s3.amazonaws.com")

	h := hmac.New(sha1.New, []byte(c.SecretAccessKey))
	h.Write([]byte(req.Method + "\n"))
	h.Write([]byte(req.Header.Get("Content-MD5") + "\n"))
	h.Write([]byte(req.Header.Get("Content-Type") + "\n"))
	h.Write([]byte(req.Header.Get("Date") + "\n"))
	h.Write(v2Headers(req))
	h.Write(v2Resource(c.Bucket, req))

	//fmt.Printf("CANONICAL:\n---\n%s\n%s\n%s\n%s\n%s%s%s]---\n",
	//	req.Method, req.Header.Get("Content-MD5"), req.Header.Get("Content-Type"), req.Header.Get("Date"), string(v2Headers(req)), v2Resource(c.Bucket, req))

	//fmt.Printf("AWS %s:%s\n", c.AccessKeyID, base64(h.Sum(nil)))
	return fmt.Sprintf("AWS %s:%s", c.AccessKeyID, base64(h.Sum(nil)))
}

func v4Headers(req *http.Request) ([]byte, []byte) {
	subset := make(map[string]string)
	names := make([]string, 0)

	for header := range req.Header {
		lc := strings.ToLower(header)
		if lc == "host" || strings.HasPrefix(lc, "x-amz-") {
			names = append(names, lc)
			subset[lc] = strings.Trim(req.Header.Get(header), " \t\r\n\f")
		}
	}
	sort.Strings(names)

	ll := make([][]byte, len(names))
	nn := make([][]byte, len(names))
	for i, header := range names {
		nn[i] = []byte(header)
		ll[i] = bytes.Join([][]byte{nn[i], []byte(subset[header])}, []byte{0x3a})
	}

	signed := bytes.Join(nn, []byte{0x3b})
	return signed, bytes.Join([][]byte{
		bytes.Join(ll, []byte{0x0a}),
		nil, /* force an empty line */
		signed,
	}, []byte{0x0a})
}

func v4QueryString(s string) []byte {
	if s == "" {
		return []byte{}
	}

	qq := strings.Split(s, "&")
	sort.Strings(qq)

	ll := make([][]byte, len(qq))
	for i := range qq {
		kv := strings.SplitN(qq[i], "=", 2)
		k := uriencode(kv[0], true)
		var v []byte
		if len(kv) == 2 {
			v = uriencode(kv[1], true)
		}

		ll[i] = make([]byte, len(k)+1+len(v))
		copy(ll[i], k)
		ll[i][len(k)] = 0x3d
		copy(ll[i][1+len(k):], v)
	}

	return bytes.Join(ll, []byte{0x26})
}

func (c *Client) v4signature(req *http.Request, raw []byte) string {
	/* step 0: assemble some temporary values we will need */
	now := time.Now().UTC()
	yyyymmdd := now.Format("20060102")
	scope := fmt.Sprintf("%s/%s/s3/aws4_request", yyyymmdd, c.Region)
	req.Header.Set("x-amz-date", now.Format("20060102T150405Z"))
	req.Header.Set("host", regexp.MustCompile(`:.*`).ReplaceAllString(req.URL.Host, ""))
	//req.Header.Set("host", "go-s3-bd6cf051-8023-4d2b-8bf2-7aaa477862ea.s3.amazonaws.com")

	payload := sha256.New()
	payload.Write(raw)
	hashed := hex(payload.Sum(nil))
	req.Header.Set("x-amz-content-sha256", hashed)

	/* step 1: generate the CanonicalRequest (+sha256() it)

	   METHOD \n
	   uri() \n
	   querystring() \n
	   headers() \n
	   signed() \n
	   payload()
	*/

	headers, hsig := v4Headers(req)
	canon := sha256.New()
	canon.Write([]byte(req.Method))
	canon.Write([]byte("\n"))
	canon.Write(uriencode(req.URL.Path, false))
	canon.Write([]byte("\n"))
	canon.Write(v4QueryString(req.URL.RawQuery))
	canon.Write([]byte("\n"))
	canon.Write(hsig)
	canon.Write([]byte("\n"))
	canon.Write([]byte(hashed))

	//fmt.Printf("CANONICAL:\n---\n%s\n%s\n%s\n%s\n%s]---\n",
	//	req.Method, string(uriencode(req.URL.Path, false)), string(v4QueryString(req.URL.RawQuery)), string(hsig), hashed)

	/* step 2: generate the StringToSign

	   AWS4-HMAC-SHA256 \n
	   YYYYMMDDTHHMMSSZ \n
	   "yyyymmdd/region/s3/aws_request" \n
	   hex(sha256(canonical()))
	*/
	cleartext := "AWS4-HMAC-SHA256" +
		"\n" + now.Format("20060102T150405Z") +
		"\n" + scope +
		"\n" + hex(canon.Sum(nil))

	//fmt.Printf("CLEARTEXT:\n---\n%s\n---\n", cleartext)

	/* step 3: generate the Signature

	   datekey = hmac-sha256("AWS4" + secret_key, YYYYMMDD)
	   datereg = hmac-sha256(datekey, region)
	   drsvc   = hmac-sha256(datereg, "s3")
	   sigkey  = hmac-sha256(drsvc, "aws4_request")

	   hex(hmac-sha256(sigkey, cleartext))

	*/
	k1 := mac256([]byte("AWS4"+c.SecretAccessKey), []byte(yyyymmdd))
	k2 := mac256(k1, []byte(c.Region))
	k3 := mac256(k2, []byte("s3"))
	k4 := mac256(k3, []byte("aws4_request"))
	sig := hex(mac256(k4, []byte(cleartext)))

	/* step 4: assemble and return the Authorize: header */
	return "AWS4-HMAC-SHA256" +
		" " + fmt.Sprintf("Credential=%s/%s", c.AccessKeyID, scope) +
		"," + fmt.Sprintf("SignedHeaders=%s", string(headers)) +
		"," + fmt.Sprintf("Signature=%s", sig)
}
