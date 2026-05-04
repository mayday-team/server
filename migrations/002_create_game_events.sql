-- +goose Up
CREATE TABLE IF NOT EXISTS game_events (
    id UUID PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES game_sessions(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    server_tick BIGINT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_game_events_session_tick ON game_events (session_id, server_tick);
CREATE INDEX IF NOT EXISTS idx_game_events_type ON game_events (type);

-- +goose Down
DROP INDEX IF EXISTS idx_game_events_type;
DROP INDEX IF EXISTS idx_game_events_session_tick;
DROP TABLE IF EXISTS game_events;
