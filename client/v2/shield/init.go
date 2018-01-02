package shield

import "bytes"

func (c *Client) Initialize(master string) (string, error) {
	in := struct {
		Master string `json:"master"`
	}{
		Master: master,
	}

	out := struct {
		DisasterKey string `json:"disaster_key"`
	}{
		DisasterKey: "",
	}

	err := c.post("/v2/init", in, &out)

	return out.DisasterKey, err
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
