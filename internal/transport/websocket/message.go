package websocket

import "github.com/mayday-team/server/internal/protocol"

// Message is a thin alias used inside the transport layer so other files
// can refer to a single name.
type Message = protocol.ServerMessage
