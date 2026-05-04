package storage

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionStartRecord struct {
	ID         string
	PlayerName string
	StartedAt  time.Time
}

type SessionEndRecord struct {
	ID                string
	EndedAt           time.Time
	SurvivedMs        int64
	FinalPhase        string
	DefeatReason      string
	ShotsFired        int
	ShotsHit          int
	DamageTaken       int
	TroopsNeutralized int
}

type SessionRepository interface {
	Start(ctx context.Context, r SessionStartRecord) error
	End(ctx context.Context, r SessionEndRecord) error
}

type PostgresSessionRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresSessionRepository(pool *pgxpool.Pool) *PostgresSessionRepository {
	return &PostgresSessionRepository{pool: pool}
}

func (r *PostgresSessionRepository) Start(ctx context.Context, rec SessionStartRecord) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO game_sessions (id, player_name, started_at)
		 VALUES ($1, $2, $3)`,
		rec.ID, rec.PlayerName, rec.StartedAt,
	)
	return err
}

func (r *PostgresSessionRepository) End(ctx context.Context, rec SessionEndRecord) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE game_sessions
		 SET ended_at = $2,
		     survived_ms = $3,
		     final_phase = $4,
		     defeat_reason = $5,
		     shots_fired = $6,
		     shots_hit = $7,
		     damage_taken = $8,
		     troops_neutralized = $9
		 WHERE id = $1`,
		rec.ID, rec.EndedAt, rec.SurvivedMs, rec.FinalPhase, rec.DefeatReason,
		rec.ShotsFired, rec.ShotsHit, rec.DamageTaken, rec.TroopsNeutralized,
	)
	return err
}

// NoopSessionRepository discards calls. Used when the DB is offline.
type NoopSessionRepository struct{}

func (NoopSessionRepository) Start(_ context.Context, _ SessionStartRecord) error { return nil }
func (NoopSessionRepository) End(_ context.Context, _ SessionEndRecord) error     { return nil }

// MemorySessionRepository is a simple in-memory store used by tests.
type MemorySessionRepository struct {
	mu      sync.Mutex
	starts  []SessionStartRecord
	ends    []SessionEndRecord
}

func NewMemorySessionRepository() *MemorySessionRepository {
	return &MemorySessionRepository{}
}

func (m *MemorySessionRepository) Start(_ context.Context, r SessionStartRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.starts = append(m.starts, r)
	return nil
}

func (m *MemorySessionRepository) End(_ context.Context, r SessionEndRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ends = append(m.ends, r)
	return nil
}

func (m *MemorySessionRepository) Starts() []SessionStartRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]SessionStartRecord, len(m.starts))
	copy(out, m.starts)
	return out
}

func (m *MemorySessionRepository) Ends() []SessionEndRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]SessionEndRecord, len(m.ends))
	copy(out, m.ends)
	return out
}
