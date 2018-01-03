package shield

import "bytes"

func (c *Client) Initialize(master string) (string, error) {
	in := struct {
		Master string `json:"master"`
	}{
		Master: master,
	}

	out := struct {
		FixedKey string `json:"fixed_key"`
	}{}

	err := c.post("/v2/init", in, &out)

	return out.FixedKey, err
}

func (c *Client) SplitKey(s string, n int) string {
	var buffer bytes.Buffer
	for i, rune := range s {
		buffer.WriteRune(rune)
		if i%n == (n-1) && i != (len(s)-1) {
			buffer.WriteRune('\n')
		}
	}
	return buffer.String()
}
