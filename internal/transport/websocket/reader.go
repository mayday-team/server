package websocket

import (
	"errors"
	"net"

	"github.com/gorilla/websocket"
	"github.com/mayday-team/server/internal/protocol"
)

// runReader pumps inbound frames into onMessage. It exits when the
// connection closes or any error is encountered. The caller is expected to
// also call client.Close() afterward, but we close defensively here too.
func (h *Handler) runReader(c *Client, onMessage func(protocol.ClientMessage)) {
	defer c.Close()

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			if !isExpectedCloseError(err) {
				h.log.Warn("ws read error", "err", err)
			}
			return
		}
		msg, perr := protocol.Parse(raw)
		if perr != nil {
			h.log.Debug("ws parse error", "err", perr)
			c.Send(protocol.ServerMessage{
				Type: protocol.ServerMsgError,
				Payload: protocol.ErrorPayload{
					Code:    "parse_error",
					Message: perr.Error(),
				},
			})
			continue
		}
		onMessage(msg)
		if h.metrics != nil {
			h.metrics.WSMessagesIn.Add(1)
		}
	}
}

func isExpectedCloseError(err error) bool {
	if errors.Is(err, net.ErrClosed) {
		return true
	}
	if websocket.IsCloseError(err,
		websocket.CloseNormalClosure,
		websocket.CloseGoingAway,
		websocket.CloseAbnormalClosure,
	) {
		return true
	}
	return false
}
