package protocol

import (
	"encoding/json"
	"fmt"
)

// ClientMessage is the discriminated union of decoded client payloads.
// Exactly one of the typed pointer fields is set for a successfully parsed
// message; the Type field always carries the wire type for logging.
type ClientMessage struct {
	Type string

	StartSession *StartSessionPayload
	PlayerInput  *PlayerInputPayload
	PlayerLook   *PlayerLookPayload
	Shoot        *ShootPayload
	Reload       *ReloadPayload
	Interact     *InteractPayload
	Ping         *PingPayload
}

// Parse decodes a raw WebSocket frame into a ClientMessage. It returns an
// error if the JSON envelope is malformed, the type is unknown, or the
// inner payload fails to decode for that type.
func Parse(raw []byte) (ClientMessage, error) {
	if len(raw) == 0 {
		return ClientMessage{}, ErrEmptyMessage
	}
	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return ClientMessage{}, fmt.Errorf("%w: %v", ErrInvalidJSON, err)
	}
	if env.Type == "" {
		return ClientMessage{}, ErrEmptyMessage
	}

	msg := ClientMessage{Type: env.Type}
	if len(env.Payload) == 0 {
		env.Payload = json.RawMessage("{}")
	}

	switch env.Type {
	case ClientMsgStartSession:
		var p StartSessionPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return ClientMessage{}, fmt.Errorf("%w: %v", ErrMalformedPayload, err)
		}
		msg.StartSession = &p
	case ClientMsgPlayerInput:
		var p PlayerInputPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return ClientMessage{}, fmt.Errorf("%w: %v", ErrMalformedPayload, err)
		}
		msg.PlayerInput = &p
	case ClientMsgPlayerLook:
		var p PlayerLookPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return ClientMessage{}, fmt.Errorf("%w: %v", ErrMalformedPayload, err)
		}
		msg.PlayerLook = &p
	case ClientMsgShoot:
		var p ShootPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return ClientMessage{}, fmt.Errorf("%w: %v", ErrMalformedPayload, err)
		}
		msg.Shoot = &p
	case ClientMsgReload:
		var p ReloadPayload
		_ = json.Unmarshal(env.Payload, &p)
		msg.Reload = &p
	case ClientMsgInteract:
		var p InteractPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return ClientMessage{}, fmt.Errorf("%w: %v", ErrMalformedPayload, err)
		}
		msg.Interact = &p
	case ClientMsgPing:
		var p PingPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return ClientMessage{}, fmt.Errorf("%w: %v", ErrMalformedPayload, err)
		}
		msg.Ping = &p
	default:
		return ClientMessage{}, fmt.Errorf("%w: %s", ErrUnknownMessage, env.Type)
	}
	return msg, nil
}

// Encode serializes a server-bound message to JSON bytes ready for the
// WebSocket layer to send.
func Encode(msg ServerMessage) ([]byte, error) {
	return json.Marshal(msg)
}
