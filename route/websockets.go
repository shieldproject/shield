package route

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jhunt/go-log"
)

type WebSocket struct {
	conn      *websocket.Conn
	writeLock sync.Mutex
	timeout   time.Duration
}

type WebSocketSettings struct {
	WriteTimeout time.Duration
}

func (r *Request) Upgrade(settings WebSocketSettings) *WebSocket {
	log.Debugf("%s upgrading to WebSockets", r)

	pongChan := make(chan bool, 1)
	pongChan <- true

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(r.w, r.Req, nil)
	if err != nil {
		r.Fail(Oops(err, "an unknown error has occurred"))
		return nil
	}

	/* track that we are "responding" with a websocket upgrade */
	r.bt = append(r.bt, "Upgrade")

	ret := &WebSocket{
		conn:    conn,
		timeout: settings.WriteTimeout,
	}

	return ret
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

	ws.writeLock.Lock()
	err = ws.conn.WriteMessage(websocket.TextMessage, b)
	ws.writeLock.Unlock()
	return websocket.IsUnexpectedCloseError(err), err
}

func (ws *WebSocket) Ping() error {
	ws.writeLock.Lock()
	defer ws.writeLock.Unlock()
	return ws.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(ws.timeout))
}

func (ws *WebSocket) SendClose() error {
	ws.writeLock.Lock()
	defer ws.writeLock.Unlock()
	return ws.conn.WriteControl(websocket.CloseMessage, nil, time.Now().Add(ws.timeout))
}
