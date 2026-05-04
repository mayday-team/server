-- +goose Up
CREATE TABLE IF NOT EXISTS game_sessions (
    id UUID PRIMARY KEY,
    player_name TEXT NOT NULL DEFAULT 'anonymous',
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ,
    survived_ms BIGINT NOT NULL DEFAULT 0,
    final_phase TEXT,
    defeat_reason TEXT,
    shots_fired INT NOT NULL DEFAULT 0,
    shots_hit INT NOT NULL DEFAULT 0,
    damage_taken INT NOT NULL DEFAULT 0,
    troops_neutralized INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_game_sessions_started_at ON game_sessions (started_at);

-- +goose Down
DROP INDEX IF EXISTS idx_game_sessions_started_at;
DROP TABLE IF EXISTS game_sessions;
