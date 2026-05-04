package config

import (
	"fmt"
	"os"
	"strconv"
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

	PlayerMaxHP        int
	PlayerMaxAmmo      int
	PlayerMoveSpeed    float64
	PlayerShootDamage  int

	ShootMaxDistance    float64
	ShootAngleThreshold float64
	FireRateLimit       time.Duration

	SessionMaxDuration time.Duration
	FinalStandAfter    time.Duration
	ForceDefeatAfter   time.Duration

	SessionEventBufferSize int
	ClientSendBufferSize   int
}

func Load() Config {
	return Config{
		Port:        getString("PORT", "3001"),
		DatabaseURL: getString("DATABASE_URL", "postgres://mayday:mayday@localhost:5432/mayday?sslmode=disable"),

		TickRate:     getInt("TICK_RATE", 30),
		SnapshotRate: getInt("SNAPSHOT_RATE", 15),

		InitialTroopCount:   getInt("INITIAL_TROOP_COUNT", 4),
		MaxTroopCount:       getInt("MAX_TROOP_COUNT", 30),
		TroopDetectionRange: getFloat("TROOP_DETECTION_RANGE", 35),
		TroopAttackRange:    getFloat("TROOP_ATTACK_RANGE", 22),
		TroopDamage:         getInt("TROOP_DAMAGE", 8),
		TroopMoveSpeed:      getFloat("TROOP_MOVE_SPEED", 4),

		PlayerMaxHP:       getInt("PLAYER_MAX_HP", 100),
		PlayerMaxAmmo:     getInt("PLAYER_MAX_AMMO", 24),
		PlayerMoveSpeed:   getFloat("PLAYER_MOVE_SPEED", 6),
		PlayerShootDamage: getInt("PLAYER_SHOOT_DAMAGE", 25),

		ShootMaxDistance:    getFloat("SHOOT_MAX_DISTANCE", 60),
		ShootAngleThreshold: getFloat("SHOOT_ANGLE_THRESHOLD", 0.96),
		FireRateLimit:       time.Duration(getInt("FIRE_RATE_LIMIT_MS", 250)) * time.Millisecond,

		SessionMaxDuration: time.Duration(getInt("SESSION_MAX_DURATION_MS", 600000)) * time.Millisecond,
		FinalStandAfter:    time.Duration(getInt("FINAL_STAND_AFTER_MS", 300000)) * time.Millisecond,
		ForceDefeatAfter:   time.Duration(getInt("FORCE_DEFEAT_AFTER_MS", 420000)) * time.Millisecond,

		SessionEventBufferSize: getInt("SESSION_EVENT_BUFFER_SIZE", 512),
		ClientSendBufferSize:   getInt("CLIENT_SEND_BUFFER_SIZE", 64),
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
	if c.SessionEventBufferSize <= 0 || c.ClientSendBufferSize <= 0 {
		return fmt.Errorf("buffer sizes must be > 0")
	}
	return nil
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
