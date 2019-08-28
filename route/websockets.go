package route

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jhunt/go-log"
)

type WebSocket struct {
	conn    *websocket.Conn
	timeout time.Duration
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

func (ws *WebSocket) Discard(onclose func()) {
	for {
		if _, _, err := ws.conn.NextReader(); err != nil {
			log.Infof("discarding message from ws client...")
			ws.conn.Close()
			break
		}
	}
	onclose()
}

func (ws *WebSocket) Write(b []byte) (bool, error) {
	err := ws.conn.SetWriteDeadline(time.Now().Add(ws.timeout))
	if err != nil {
		return true, err
	}
	err = ws.conn.WriteMessage(websocket.TextMessage, b)
	return websocket.IsUnexpectedCloseError(err), err
}

func (ws *WebSocket) SetWriteTimeout(timeout time.Duration) {
	ws.timeout = timeout
}

func (ws *WebSocket) SendClose() error {
	return ws.conn.WriteControl(websocket.CloseMessage, nil, time.Now().Add(ws.timeout))
}
