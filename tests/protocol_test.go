package tests

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mayday-team/server/internal/protocol"
)

func TestParseStartSession(t *testing.T) {
	raw := []byte(`{"type":"start_session","payload":{"player_name":"jin"}}`)
	msg, err := protocol.Parse(raw)
	require.NoError(t, err)
	require.NotNil(t, msg.StartSession)
	assert.Equal(t, "jin", msg.StartSession.PlayerName)
	assert.Equal(t, "start_session", msg.Type)
}

func TestParsePlayerInput(t *testing.T) {
	raw := []byte(`{"type":"player_input","payload":{"seq":12,"move":{"forward":true,"right":true},"delta_ms":16}}`)
	msg, err := protocol.Parse(raw)
	require.NoError(t, err)
	require.NotNil(t, msg.PlayerInput)
	assert.Equal(t, int64(12), msg.PlayerInput.Seq)
	assert.True(t, msg.PlayerInput.Move.Forward)
	assert.True(t, msg.PlayerInput.Move.Right)
	assert.False(t, msg.PlayerInput.Move.Backward)
	assert.Equal(t, int64(16), msg.PlayerInput.DeltaMs)
}

func TestParseShoot(t *testing.T) {
	raw := []byte(`{"type":"shoot","payload":{"seq":1,"origin":{"x":0,"y":1.6,"z":0},"direction":{"x":0,"y":0,"z":1},"client_time":42}}`)
	msg, err := protocol.Parse(raw)
	require.NoError(t, err)
	require.NotNil(t, msg.Shoot)
	assert.InDelta(t, 1.6, msg.Shoot.Origin.Y, 1e-9)
	assert.InDelta(t, 1.0, msg.Shoot.Direction.Z, 1e-9)
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := protocol.Parse([]byte(`not json`))
	assert.ErrorIs(t, err, protocol.ErrInvalidJSON)
}

func TestParseUnknownType(t *testing.T) {
	_, err := protocol.Parse([]byte(`{"type":"telekinesis","payload":{}}`))
	assert.ErrorIs(t, err, protocol.ErrUnknownMessage)
}

func TestParseEmptyMessage(t *testing.T) {
	_, err := protocol.Parse([]byte(``))
	assert.True(t, errors.Is(err, protocol.ErrEmptyMessage))

	_, err = protocol.Parse([]byte(`{"type":"","payload":{}}`))
	assert.True(t, errors.Is(err, protocol.ErrEmptyMessage))
}

func TestParseMalformedPayload(t *testing.T) {
	raw := []byte(`{"type":"player_input","payload":{"seq":"not-a-number"}}`)
	_, err := protocol.Parse(raw)
	assert.ErrorIs(t, err, protocol.ErrMalformedPayload)
}

func TestEncodeServerMessage(t *testing.T) {
	msg := protocol.ServerMessage{
		Type:    protocol.ServerMsgPong,
		Payload: protocol.PongPayload{ClientTime: 1, ServerTime: 2},
	}
	body, err := protocol.Encode(msg)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"type":"pong"`)
	assert.Contains(t, string(body), `"client_time":1`)
}
