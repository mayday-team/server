package observability

import (
	"sync/atomic"
	"time"
)

type Metrics struct {
	StartTime         time.Time
	ActiveSessions    atomic.Int64
	TotalSessions     atomic.Int64
	TickCount         atomic.Int64
	SnapshotCount     atomic.Int64
	WSMessagesIn      atomic.Int64
	WSMessagesOut     atomic.Int64
	ShotsFiredTotal   atomic.Int64
	ShotsHitTotal     atomic.Int64
	DefeatsTotal      atomic.Int64
}

func New() *Metrics {
	return &Metrics{StartTime: time.Now()}
}

func (m *Metrics) UptimeSeconds() int64 {
	return int64(time.Since(m.StartTime).Seconds())
}
