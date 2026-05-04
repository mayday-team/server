package websocket

import (
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mayday-team/server/internal/protocol"
)

// Client is the per-connection wrapper that bridges a WebSocket conn to a
// game session. Each Client owns:
//   - a buffered outbound channel that the writer goroutine consumes,
//   - a dedicated reader goroutine that decodes inbound frames.
type Client struct {
	conn   *websocket.Conn
	sendCh chan protocol.ServerMessage
	log    *slog.Logger

	closeOnce sync.Once
	closed    chan struct{}

	writeDeadline time.Duration
}

func newClient(conn *websocket.Conn, log *slog.Logger, sendBuffer int) *Client {
	if sendBuffer <= 0 {
		sendBuffer = 64
	}
	return &Client{
		conn:          conn,
		sendCh:        make(chan protocol.ServerMessage, sendBuffer),
		log:           log,
		closed:        make(chan struct{}),
		writeDeadline: 2 * time.Second,
	}
}

// Send enqueues a message for the writer goroutine. It is non-blocking: if
// the buffer is full the message is dropped, which is the right behavior
// for a single-player server where snapshots arrive every tick anyway.
func (c *Client) Send(msg protocol.ServerMessage) {
	select {
	case c.sendCh <- msg:
	case <-c.closed:
	default:
		c.log.Warn("client send buffer full; dropping message", "type", msg.Type)
	}
}

// Close signals the reader and writer goroutines to exit and closes the
// underlying connection. Safe to call multiple times.
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.closed)
		_ = c.conn.Close()
	})
}

// Closed returns a channel closed once the client has been shut down.
func (c *Client) Closed() <-chan struct{} { return c.closed }
