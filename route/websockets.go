package route

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/jhunt/go-log"
)

type WebSocket struct {
	conn *websocket.Conn
}

func (r *Request) Upgrade() *WebSocket {
	log.Debugf("%s upgrading to WebSockets", r)

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(r.w, r.Req, nil)
	if err != nil {
		r.Fail(Oops(err, "an unknown error has occurred"))
		return nil
	}

	return &WebSocket{
		conn: conn,
	}
}

func (ws *WebSocket) Discard() {
	for {
		if _, _, err := ws.conn.NextReader(); err != nil {
			log.Infof("discarding message from ws client...")
			ws.conn.Close()
			break
		}
	}
}

func (ws *WebSocket) Write(b []byte) error {
	return ws.conn.WriteMessage(websocket.TextMessage, b)
}
