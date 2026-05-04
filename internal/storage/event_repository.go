package storage

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// EventRecord is the persistence-shape of a session event row.
type EventRecord struct {
	ID         string
	SessionID  string
	Type       string
	ServerTick int64
	Payload    []byte
	CreatedAt  time.Time
}

// EventRepository is the interface the game session writes events through.
// We expose an interface so the session can run without a database (the
// no-op repo) which keeps the server bootable even when Postgres is down.
type EventRepository interface {
	Insert(ctx context.Context, e EventRecord) error
}

// PostgresEventRepository persists events to the game_events table.
type PostgresEventRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresEventRepository(pool *pgxpool.Pool) *PostgresEventRepository {
	return &PostgresEventRepository{pool: pool}
}

func (r *PostgresEventRepository) Insert(ctx context.Context, e EventRecord) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO game_events (id, session_id, type, server_tick, payload, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		e.ID, e.SessionID, e.Type, e.ServerTick, e.Payload, e.CreatedAt,
	)
	return err
}

// NoopEventRepository is used when no database is configured. It silently
// drops events so the rest of the system can keep running.
type NoopEventRepository struct{}

func (NoopEventRepository) Insert(_ context.Context, _ EventRecord) error { return nil }

// MemoryEventRepository keeps events in memory. Useful for tests.
type MemoryEventRepository struct {
	mu     sync.Mutex
	events []EventRecord
}

func NewMemoryEventRepository() *MemoryEventRepository {
	return &MemoryEventRepository{}
}

func (m *MemoryEventRepository) Insert(_ context.Context, e EventRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, e)
	return nil
}

func (m *MemoryEventRepository) All() []EventRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]EventRecord, len(m.events))
	copy(out, m.events)
	return out
}
