package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port        string
	DatabaseURL string

	TickRate     int
	SnapshotRate int

	InitialTroopCount   int
	MaxTroopCount       int
	TroopDetectionRange float64
	TroopAttackRange    float64
	TroopDamage         int
	TroopMoveSpeed      float64
	TroopBaseAccuracy   float64

	PlayerMaxHP       int
	PlayerMaxAmmo     int
	PlayerMoveSpeed   float64
	PlayerShootDamage int

	ShootMaxDistance    float64
	ShootAngleThreshold float64
	FireRateLimit       time.Duration

	SessionMaxDuration time.Duration
	FinalStandAfter    time.Duration
	ForceDefeatAfter   time.Duration

	SessionEventBufferSize int
	ClientSendBufferSize   int

	MaxSessions        int
	ClientReadTimeout  time.Duration
	ClientPingInterval time.Duration
	ClientWriteTimeout time.Duration
	AllowedOrigins     []string
}

func Load() Config {
	return Config{
		Port:        getString("PORT", "3001"),
		DatabaseURL: getString("DATABASE_URL", "postgres://mayday:mayday@localhost:5432/mayday?sslmode=disable"),

		TickRate:     getInt("TICK_RATE", 30),
		SnapshotRate: getInt("SNAPSHOT_RATE", 15),

		InitialTroopCount:   getInt("INITIAL_TROOP_COUNT", 4),
		MaxTroopCount:       getInt("MAX_TROOP_COUNT", 45),
		TroopDetectionRange: getFloat("TROOP_DETECTION_RANGE", 35),
		TroopAttackRange:    getFloat("TROOP_ATTACK_RANGE", 22),
		TroopDamage:         getInt("TROOP_DAMAGE", 4),
		TroopMoveSpeed:      getFloat("TROOP_MOVE_SPEED", 4),
		TroopBaseAccuracy:   getFloat("TROOP_BASE_ACCURACY", 0.55),

		PlayerMaxHP:       getInt("PLAYER_MAX_HP", 1000),
		PlayerMaxAmmo:     getInt("PLAYER_MAX_AMMO", 24),
		PlayerMoveSpeed:   getFloat("PLAYER_MOVE_SPEED", 8),
		PlayerShootDamage: getInt("PLAYER_SHOOT_DAMAGE", 60),

		ShootMaxDistance:    getFloat("SHOOT_MAX_DISTANCE", 60),
		ShootAngleThreshold: getFloat("SHOOT_ANGLE_THRESHOLD", 0.96),
		FireRateLimit:       time.Duration(getInt("FIRE_RATE_LIMIT_MS", 250)) * time.Millisecond,

		SessionMaxDuration: time.Duration(getInt("SESSION_MAX_DURATION_MS", 600000)) * time.Millisecond,
		FinalStandAfter:    time.Duration(getInt("FINAL_STAND_AFTER_MS", 300000)) * time.Millisecond,
		ForceDefeatAfter:   time.Duration(getInt("FORCE_DEFEAT_AFTER_MS", 420000)) * time.Millisecond,

		SessionEventBufferSize: getInt("SESSION_EVENT_BUFFER_SIZE", 512),
		ClientSendBufferSize:   getInt("CLIENT_SEND_BUFFER_SIZE", 64),

		MaxSessions:        getInt("MAX_SESSIONS", 100),
		ClientReadTimeout:  time.Duration(getInt("CLIENT_READ_TIMEOUT_MS", 60000)) * time.Millisecond,
		ClientPingInterval: time.Duration(getInt("CLIENT_PING_INTERVAL_MS", 25000)) * time.Millisecond,
		ClientWriteTimeout: time.Duration(getInt("CLIENT_WRITE_TIMEOUT_MS", 2000)) * time.Millisecond,
		AllowedOrigins:     splitCSV(getString("ALLOWED_ORIGINS", "")),
	}
}

func (c Config) Validate() error {
	if c.TickRate <= 0 {
		return fmt.Errorf("TICK_RATE must be > 0")
	}
	if c.SnapshotRate <= 0 || c.SnapshotRate > c.TickRate {
		return fmt.Errorf("SNAPSHOT_RATE must be > 0 and <= TICK_RATE")
	}
	if c.PlayerMaxHP <= 0 {
		return fmt.Errorf("PLAYER_MAX_HP must be > 0")
	}
	if c.TroopBaseAccuracy < 0 || c.TroopBaseAccuracy > 1 {
		return fmt.Errorf("TROOP_BASE_ACCURACY must be in [0, 1]")
	}
	if c.SessionEventBufferSize <= 0 || c.ClientSendBufferSize <= 0 {
		return fmt.Errorf("buffer sizes must be > 0")
	}
	if c.MaxSessions <= 0 {
		return fmt.Errorf("MAX_SESSIONS must be > 0")
	}
	if c.ClientPingInterval >= c.ClientReadTimeout {
		return fmt.Errorf("CLIENT_PING_INTERVAL must be < CLIENT_READ_TIMEOUT")
	}
	return nil
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getString(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getFloat(k string, def float64) float64 {
	if v := os.Getenv(k); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}
