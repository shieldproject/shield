package vault

import (
	"crypto/aes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

type Parameters struct {
	Key  string `json:"key"`
	IV   string `json:"iv"`
	Type string `json:"type"`
	UUID string `json:"uuid"`
}

func keygen(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

func gen(t string, k, i int) (Parameters, error) {
	key, err := keygen(k)
	if err != nil {
		return Parameters{}, err
	}

	iv, err := keygen(i)
	if err != nil {
		return Parameters{}, err
	}

	return Parameters{
		Type: t,
		Key:  key,
		IV:   iv,
	}, nil
}

func (c *Client) RandomParameters(typ string) (Parameters, error) {
	cipher := strings.Split(typ, "-")[0]
	switch cipher {
	case "aes128":
		return gen(typ, 128/8, aes.BlockSize)

	case "aes256":
		return gen(typ, 256/8, aes.BlockSize)

	default:
		return Parameters{}, fmt.Errorf("unrecognized cipher/mode '%s'", typ)
	}
}

func (c *Client) FixedParameters() (string, Parameters, error) {
	k, err := keygen(512)
	if err != nil {
		return "", Parameters{}, err
	}

	g := pbkdf2.Key([]byte(k[32:]), []byte(k), 4096, 48, sha256.New)
	return k, Parameters{
		Type: "aes256-ctr",
		Key:  hex.EncodeToString(g[:32]),
		IV:   hex.EncodeToString(g[32:]),
	}, nil
}
func (c *Client) Store(id string, params Parameters) error {
	params.UUID = id
	params.Key = Encode(params.Key, 4)
	params.IV = Encode(params.IV, 4)
	return c.Post("secret/archives/"+id, params, nil)
}

func (c *Client) Retrieve(id string) (Parameters, error) {
	var out Parameters
	ok, err := c.Get(fmt.Sprintf("secret/archives/%s", id), &out)
	if !ok {
		err = fmt.Errorf("not found in vault")
	}
	if err != nil {
		return out, fmt.Errorf("failed to retrieve encryption parameters for [%s]: %s", id, err)
	}

	out.Key = Decode(out.Key)
	out.IV = Decode(out.IV)
	return out, nil
}

func (c *Client) RetrieveFixed() (Parameters, error) {
	return c.Retrieve("fixed_key")
}

func (c *Client) StoreFixed(p Parameters) error {
	return c.Store("fixed_key", p)
}
