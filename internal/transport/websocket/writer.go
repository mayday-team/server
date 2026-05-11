package websocket

import (
	"time"

	"github.com/gorilla/websocket"
	"github.com/mayday-team/server/internal/protocol"
)

// runWriter pumps outbound messages from the client send buffer onto the
// wire. A write deadline keeps a slow consumer from stalling the session.
// It also drives server-initiated ping frames; the reader's pong handler
// extends the read deadline so an idle-but-healthy client stays connected.
func (h *Handler) runWriter(c *Client) {
	defer c.Close()
	pingEvery := h.pingInterval
	if pingEvery <= 0 {
		pingEvery = 25 * time.Second
	}
	ping := time.NewTicker(pingEvery)
	defer ping.Stop()
	for {
		select {
		case <-c.closed:
			return
		case <-ping.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(c.writeDeadline))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				h.log.Debug("ws ping error", "err", err)
				return
			}
		case msg, ok := <-c.sendCh:
			if !ok {
				return
			}
			if err := c.writeMessage(msg); err != nil {
				h.log.Debug("ws write error", "err", err)
				return
			}
			if h.metrics != nil {
				h.metrics.WSMessagesOut.Add(1)
			}
		}
	}
}

func (c *Client) writeMessage(msg protocol.ServerMessage) error {
	body, err := protocol.Encode(msg)
	if err != nil {
		return err
	}
	_ = c.conn.SetWriteDeadline(time.Now().Add(c.writeDeadline))
	return c.conn.WriteMessage(websocket.TextMessage, body)
}
