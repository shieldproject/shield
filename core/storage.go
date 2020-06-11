package core

import (
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/jhunt/go-log"
	"github.com/jhunt/ssg/pkg/client"
)

type stream struct {
	gateway string
	id      string
	token   string
	path    string
}

func (c *Core) gateways() []client.Client {
	choices := make([]string, len(c.Config.StorageGateway.Gateways))
	for i, url := range c.Config.StorageGateway.Gateways {
		choices[i] = url
	}
	l := sort.StringSlice(choices)
	rand.Shuffle(l.Len(), l.Swap)

	clients := make([]client.Client, len(l))
	for i, url := range l {
		clients[i] = client.Client{
			URL:          url,
			ControlToken: c.Config.StorageGateway.Token,
		}
	}
	return clients
}

func (c *Core) upload(bucket, uuid string) (*stream, error) {
	t := time.Now()
	y, m, d := t.Date()
	hh, mm, ss := t.Clock()
	// FIXME: testdev is hard-coded in upload()
	p := fmt.Sprintf("ssg://testdev/%s/%04d/%02d/%02d/%04d-%02d-%02d-%02d%02d%02d-%s", bucket, y, m, d, y, m, d, hh, mm, ss, uuid)

	for _, gw := range c.gateways() {
		up, err := gw.NewUpload(p)
		if err != nil {
			log.Errorf("unable to start upload via storage gateway %s: %s", gw.URL, err)
			continue
		}

		return &stream{
			gateway: gw.URL,
			id:      up.ID,
			token:   up.Token,
			path:    up.Canon,
		}, nil
	}

	return nil, fmt.Errorf("no storage gateways could be reached")
}

func (c *Core) download(from string) (*stream, error) {
	for _, gw := range c.gateways() {
		up, err := gw.NewDownload(from)
		if err != nil {
			log.Errorf("unable to start download via storage gateway %s: %s", gw.URL, err)
			continue
		}

		return &stream{
			gateway: gw.URL,
			id:      up.ID,
			token:   up.Token,
			path:    up.Canon,
		}, nil
	}

	return nil, fmt.Errorf("no storage gateways could be reached")
}

func (c *Core) expunge(file string) error {
	for _, gw := range c.gateways() {
		err := gw.Expunge(file)
		if err != nil {
			log.Errorf("unable to expunge archive via storage gateway %s: %s", gw.URL, err)
			continue
		}

		return nil
	}

	return fmt.Errorf("no storage gateways could be reached")
}
