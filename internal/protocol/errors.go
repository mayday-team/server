package protocol

import "errors"

var (
	ErrInvalidJSON       = errors.New("invalid_json")
	ErrUnknownMessage    = errors.New("unknown_message_type")
	ErrMalformedPayload  = errors.New("malformed_payload")
	ErrEmptyMessage      = errors.New("empty_message")
)

// ErrorPayload is the wire shape of an error message sent back to clients.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
