package shield

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

type Event struct {
	Event string      `json:"event"`
	Queue string      `json:"queue"`
	Type  string      `json:"type,omitempty"`
	Data  interface{} `json:"data"`
}

func (c *Client) StreamEvents(fn func(Event)) error {
	var event Event

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u, err := url.Parse(c.URL + "/v2/events")
	if err != nil {
		return err
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	}

	header := make(http.Header)
	header.Set("X-Shield-Session", c.Session)

	ws, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		return err
	}
	defer ws.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, b, err := ws.ReadMessage()
			if err != nil {
				return
			}

			err = json.Unmarshal(b, &event)
			if err != nil {
				return
			}

			fn(event)
		}
	}()

	for {
		select {
		case <-done:
			return nil

		case <-interrupt:
			err := ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				return err
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return nil
		}
	}
}
