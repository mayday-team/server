package game

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/mayday-team/server/internal/ai"
	gmath "github.com/mayday-team/server/internal/game/math"
	"github.com/mayday-team/server/internal/game/scenario"
	"github.com/mayday-team/server/internal/game/state"
	"github.com/mayday-team/server/internal/game/systems"
	"github.com/mayday-team/server/internal/protocol"
)

func (s *Session) run() {
	defer close(s.doneCh)
	defer func() {
		if r := recover(); r != nil {
			s.log.Error("session loop panic", "panic", r)
		}
		s.finalize()
	}()

	s.spawnInitialTroops()

	s.recordEvent(EventSessionStarted, map[string]any{
		"session_id":  s.id,
		"player_name": s.player.Name,
		"tick_rate":   s.cfg.TickRate,
	})

	s.sendNonBlocking(protocol.ServerMessage{
		Type: protocol.ServerMsgWelcome,
		Payload: protocol.WelcomePayload{
			ServerVersion: "mayday-mvp",
			ServerTime:    time.Now().UnixMilli(),
		},
	})
	s.sendNonBlocking(protocol.ServerMessage{
		Type: protocol.ServerMsgSessionStarted,
		Payload: protocol.SessionStartedPayload{
			SessionID: s.id,
			TickRate:  s.cfg.TickRate,
			StartedAt: s.startedAt.UnixMilli(),
		},
	})

	tickInterval := time.Second / time.Duration(s.cfg.TickRate)
	if tickInterval <= 0 {
		tickInterval = 33 * time.Millisecond
	}
	snapshotEvery := s.cfg.TickRate / s.cfg.SnapshotRate
	if snapshotEvery <= 0 {
		snapshotEvery = 1
	}

	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case now := <-ticker.C:
			if s.tickStep(now, snapshotEvery) {
				return
			}
		}
	}
}

// tickStep advances the session by one tick. Returns true if the session
// has reached a terminal state and the loop should exit.
func (s *Session) tickStep(now time.Time, snapshotEvery int) (ended bool) {
	s.serverTick++
	deltaMs := now.Sub(s.lastTickAt).Milliseconds()
	s.lastTickAt = now

	s.drainInputs(now)

	upd := s.director.Tick(scenario.Input{
		Now:                 now,
		PlayerHP:            s.player.HP,
		PlayerMaxHP:         s.player.MaxHP,
		PlayerAmmo:          s.player.Ammo,
		PlayerMaxAmmo:       s.player.MaxAmmo,
		PlayerAlive:         s.player.IsAlive,
		SurvivingTroopCount: aliveTroopCount(s.troops),
	})
	s.handleDirectorUpdate(upd)

	s.maybeSpawnTroops(upd, now)

	s.runAI(now, deltaMs)
	s.applyTroopShots(now)
	s.cleanupDeadTroops()

	systems.AccumulateSurvival(s.player, deltaMs)

	if upd.TriggeredDefeat || s.director.CurrentPhase() == scenario.PhaseDefeat {
		s.broadcastSnapshot(now)
		return true
	}

	if s.serverTick%int64(snapshotEvery) == 0 {
		s.broadcastSnapshot(now)
	}
	return false
}

func (s *Session) drainInputs(now time.Time) {
	for {
		select {
		case msg := <-s.inputCh:
			s.handleInput(msg, now)
		default:
			return
		}
	}
}

func (s *Session) handleInput(msg protocol.ClientMessage, now time.Time) {
	s.player.LastSeenAt = now
	switch {
	case msg.PlayerInput != nil:
		systems.ApplyPlayerMovement(s.player, systems.MovementInput{
			Forward:  msg.PlayerInput.Move.Forward,
			Backward: msg.PlayerInput.Move.Backward,
			Left:     msg.PlayerInput.Move.Left,
			Right:    msg.PlayerInput.Move.Right,
		}, msg.PlayerInput.DeltaMs, s.cfg.PlayerMoveSpeed)
		s.player.LastProcessedInputSeq = msg.PlayerInput.Seq
	case msg.PlayerLook != nil:
		systems.ApplyPlayerLook(s.player, msg.PlayerLook.Yaw, msg.PlayerLook.Pitch)
	case msg.Shoot != nil:
		s.handleShoot(*msg.Shoot, now)
	case msg.Reload != nil:
		// MVP: reload simply tops up to MaxAmmo only if zero, with no
		// timer; keeps the player able to test the system.
		if s.player.IsAlive && s.player.Ammo == 0 {
			s.player.Ammo = s.player.MaxAmmo
		}
	case msg.Interact != nil:
		// No interactable objects in the MVP; acknowledge silently.
	case msg.Ping != nil:
		s.sendNonBlocking(protocol.ServerMessage{
			Type: protocol.ServerMsgPong,
			Payload: protocol.PongPayload{
				ClientTime: msg.Ping.ClientTime,
				ServerTime: now.UnixMilli(),
			},
		})
	}
}

func (s *Session) handleShoot(p protocol.ShootPayload, now time.Time) {
	s.stats.ShotsFired++
	cfg := systems.ShootConfig{
		MaxDistance:    s.cfg.ShootMaxDistance,
		AngleThreshold: s.cfg.ShootAngleThreshold,
		Damage:         s.cfg.PlayerShootDamage,
		FireRateLimit:  s.cfg.FireRateLimit,
	}
	out := systems.ProcessPlayerShoot(s.player, s.troops, p.Origin, p.Direction, cfg, now)

	if out.Accepted {
		s.recordEvent(EventPlayerShot, map[string]any{
			"seq":         p.Seq,
			"hit":         out.HitTroopID != "",
			"hit_troop":   out.HitTroopID,
			"hit_distance": out.HitDistance,
		})
		if out.HitTroopID != "" {
			s.stats.ShotsHit++
			s.recordEvent(EventPlayerHitTroop, map[string]any{
				"troop_id": out.HitTroopID,
				"damage":   out.DamageDealt,
				"killed":   out.TroopKilled,
			})
			if out.TroopKilled {
				s.stats.TroopsNeutralized++
			}
		}
	}

	s.sendNonBlocking(protocol.ServerMessage{
		Type: protocol.ServerMsgShotResult,
		Payload: protocol.ShotResultPayload{
			Seq:         p.Seq,
			Accepted:    out.Accepted,
			Reason:      out.Reason,
			HitTroopID:  out.HitTroopID,
			HitDistance: out.HitDistance,
			DamageDealt: out.DamageDealt,
			TroopKilled: out.TroopKilled,
			AmmoLeft:    s.player.Ammo,
		},
	})
}

func (s *Session) handleDirectorUpdate(upd scenario.Update) {
	if upd.PressureChanged {
		s.sendNonBlocking(protocol.ServerMessage{
			Type: protocol.ServerMsgPressureChanged,
			Payload: protocol.PressureChangedPayload{
				PressureLevel:     upd.PressureLevel,
				EncirclementLevel: upd.EncirclementLevel,
			},
		})
		s.recordEvent(EventPressureChanged, map[string]any{
			"pressure":     upd.PressureLevel,
			"encirclement": upd.EncirclementLevel,
		})
	}
	if upd.PhaseChanged {
		s.sendNonBlocking(protocol.ServerMessage{
			Type: protocol.ServerMsgScenarioPhaseChanged,
			Payload: protocol.ScenarioPhaseChangedPayload{
				PreviousPhase: upd.PreviousPhase,
				CurrentPhase:  upd.CurrentPhase,
				Tick:          s.serverTick,
			},
		})
		s.recordEvent(EventPhaseChanged, map[string]any{
			"previous_phase": upd.PreviousPhase,
			"current_phase":  upd.CurrentPhase,
		})
	}
	if upd.TriggeredDefeat {
		s.sendNonBlocking(protocol.ServerMessage{
			Type: protocol.ServerMsgDefeatTriggered,
			Payload: protocol.DefeatTriggeredPayload{
				Reason: upd.DefeatReason,
				Tick:   s.serverTick,
			},
		})
		s.recordEvent(EventDefeatTriggered, map[string]any{
			"reason": upd.DefeatReason,
		})
		if !s.player.IsAlive {
			s.recordEvent(EventPlayerDied, map[string]any{
				"tick": s.serverTick,
			})
		}
	}
}

func (s *Session) spawnInitialTroops() {
	for i := 0; i < s.cfg.InitialTroopCount; i++ {
		s.spawnTroopAroundPlayer(0)
	}
}

func (s *Session) maybeSpawnTroops(upd scenario.Update, _ time.Time) {
	if !upd.PhaseChanged {
		return
	}
	switch upd.CurrentPhase {
	case scenario.PhaseEscalation:
		s.spawnTroopBatch(2)
	case scenario.PhaseReinforcement:
		s.spawnTroopBatch(4)
	case scenario.PhaseEncirclement:
		s.spawnTroopBatch(5)
	case scenario.PhaseFinalStand:
		s.spawnTroopBatch(6)
	}
}

func (s *Session) spawnTroopBatch(n int) {
	for i := 0; i < n; i++ {
		if aliveTroopCount(s.troops) >= s.cfg.MaxTroopCount {
			return
		}
		s.spawnTroopAroundPlayer(0)
	}
}

func (s *Session) spawnTroopAroundPlayer(_ float64) {
	angle := s.rng.Float64() * 2 * math.Pi
	radius := s.cfg.TroopDetectionRange*0.8 + s.rng.Float64()*5
	pos := gmath.Vector3{
		X: s.player.Position.X + radius*math.Cos(angle),
		Y: 0,
		Z: s.player.Position.Z + radius*math.Sin(angle),
	}
	t := &state.MartialTroopState{
		ID:         uuid.NewString(),
		Position:   pos,
		HP:         StartingTroopHP,
		MaxHP:      StartingTroopHP,
		Ammo:       StartingTroopAmmo,
		MaxAmmo:    StartingTroopAmmo,
		State:      ai.StatePatrol,
		IsAlive:    true,
		Difficulty: "standard",
		SquadID:    "alpha",
	}
	s.troops[t.ID] = t
	s.spawnedTotal++
	s.sendNonBlocking(protocol.ServerMessage{
		Type: protocol.ServerMsgTroopSpawned,
		Payload: protocol.TroopSpawnedPayload{
			Troop:      troopToSnapshot(t),
			ServerTick: s.serverTick,
		},
	})
	s.recordEvent(EventTroopSpawned, map[string]any{
		"troop_id": t.ID,
		"position": pos,
	})
}

func (s *Session) runAI(_ time.Time, deltaMs int64) {
	minTroops := MinTroopFloor
	count := aliveTroopCount(s.troops)
	for _, t := range s.troops {
		if t == nil || !t.IsAlive {
			continue
		}
		percIn := ai.PerceptionInput{
			PlayerAlive:    s.player.IsAlive,
			PlayerPosition: s.player.Position,
			DetectionRange: s.cfg.TroopDetectionRange,
			AttackRange:    s.cfg.TroopAttackRange,
		}
		decisionIn := ai.DecisionInput{
			Troop: ai.TroopSnapshot{
				ID:       t.ID,
				Position: t.Position,
				HP:       t.HP,
				MaxHP:    t.MaxHP,
				Ammo:     t.Ammo,
				IsAlive:  t.IsAlive,
				State:    t.State,
			},
			Phase:         s.director.CurrentPhase(),
			Pressure:      s.director.PressureLevel(),
			Encirclement:  s.director.EncirclementLevel(),
			EscapeBlocked: s.director.EscapeBlocked(),
			TroopCount:    count,
			MaxTroops:     s.cfg.MaxTroopCount,
			MinTroops:     minTroops,
		}
		_, decision := s.aiCtrl.Update(t.Position, decisionIn, percIn)
		t.State = decision.NextState
		s.applyTroopActions(t, decision.Actions, deltaMs)
	}
}

func (s *Session) applyTroopActions(t *state.MartialTroopState, actions []ai.Action, deltaMs int64) {
	for _, a := range actions {
		switch a.Kind {
		case ai.ActionMoveTo, ai.ActionFlankTo, ai.ActionBlockExit:
			if a.HasPoint {
				systems.MoveTroopToward(t, a.Target, deltaMs, s.cfg.TroopMoveSpeed)
			}
		case ai.ActionLookAt:
			if a.HasPoint {
				dx := a.Target.X - t.Position.X
				dz := a.Target.Z - t.Position.Z
				t.Yaw = math.Atan2(dx, dz)
			}
		case ai.ActionShoot:
			t.LastKnownTargetPosition = clonePoint(s.player.Position)
		case ai.ActionSuppressArea:
			// Suppression has no per-action mechanical effect in MVP beyond
			// emitting an event; ammo/damage flows through the shoot path.
		case ai.ActionTakeCover:
			t.Velocity = gmath.Vector3{}
		case ai.ActionCallReinforcement:
			if aliveTroopCount(s.troops) < s.cfg.MaxTroopCount && s.spawnedTotal < s.cfg.MaxTroopCount {
				s.spawnTroopAroundPlayer(0)
			}
		}
	}
}

func (s *Session) applyTroopShots(now time.Time) {
	if !s.player.IsAlive {
		return
	}
	for _, t := range s.troops {
		if t == nil || !t.IsAlive {
			continue
		}
		if t.State != ai.StateAttack && t.State != ai.StateSuppress {
			continue
		}
		if gmath.Distance(t.Position, s.player.Position) > s.cfg.TroopAttackRange {
			continue
		}
		res, fired := systems.TroopShootAttempt(
			t, s.player,
			s.cfg.TroopDamage,
			time.Duration(TroopFireRateMs)*time.Millisecond,
			now,
		)
		if !fired {
			continue
		}
		s.stats.DamageTaken += res.AppliedDamage
		s.sendNonBlocking(protocol.ServerMessage{
			Type: protocol.ServerMsgDamageTaken,
			Payload: protocol.DamageTakenPayload{
				Source:      "martial_troop",
				SourceID:    t.ID,
				Damage:      res.AppliedDamage,
				RemainingHP: res.RemainingHP,
			},
		})
		s.recordEvent(EventPlayerDamaged, map[string]any{
			"source_id": t.ID,
			"damage":    res.AppliedDamage,
			"hp_left":   res.RemainingHP,
		})
		if res.Killed {
			s.sendNonBlocking(protocol.ServerMessage{
				Type:    protocol.ServerMsgPlayerDied,
				Payload: protocol.PlayerDiedPayload{SessionID: s.id, Tick: s.serverTick},
			})
			s.recordEvent(EventPlayerDied, map[string]any{"tick": s.serverTick})
			return
		}
	}
}

func (s *Session) cleanupDeadTroops() {
	for id, t := range s.troops {
		if t != nil && !t.IsAlive {
			delete(s.troops, id)
		}
	}
}

func (s *Session) broadcastSnapshot(now time.Time) {
	troopList := make([]protocol.TroopSnapshot, 0, len(s.troops))
	for _, t := range s.troops {
		troopList = append(troopList, troopToSnapshot(t))
	}
	events := s.pendingEventSnapshots
	s.pendingEventSnapshots = s.pendingEventSnapshots[:0]
	_ = now
	s.sendNonBlocking(protocol.ServerMessage{
		Type: protocol.ServerMsgStateSnapshot,
		Payload: protocol.StateSnapshotPayload{
			ServerTick:        s.serverTick,
			SessionID:         s.id,
			ScenarioPhase:     s.director.CurrentPhase(),
			PressureLevel:     s.director.PressureLevel(),
			EncirclementLevel: s.director.EncirclementLevel(),
			Player:            playerToSnapshot(s.player),
			Troops:            troopList,
			Events:            events,
		},
	})
}

func (s *Session) recordEvent(t EventType, payload any) {
	ev := NewEvent(s.id, t, s.serverTick, payload)
	s.pendingEventSnapshots = append(s.pendingEventSnapshots, protocol.EventSnapshot{
		Type:       string(t),
		ServerTick: s.serverTick,
	})
	if len(s.pendingEventSnapshots) > SnapshotEventLimit {
		s.pendingEventSnapshots = s.pendingEventSnapshots[len(s.pendingEventSnapshots)-SnapshotEventLimit:]
	}
	s.stats.EventsRecorded++
	s.sendNonBlocking(protocol.ServerMessage{
		Type: protocol.ServerMsgEventLogged,
		Payload: protocol.EventLoggedPayload{
			Type:       string(t),
			ServerTick: s.serverTick,
		},
	})
	rec := storage.EventRecord{
		ID:         ev.ID,
		SessionID:  ev.SessionID,
		Type:       string(ev.Type),
		ServerTick: ev.ServerTick,
		Payload:    []byte(ev.Payload),
		CreatedAt:  ev.CreatedAt,
	}
	select {
	case s.eventBuf <- rec:
	default:
		s.log.Warn("event buffer full; dropping for persistence", "type", t)
	}
}

func (s *Session) finalize() {
	close(s.eventBuf)

	endedAt := time.Now()
	finalPhase := s.director.CurrentPhase()
	defeat := s.director.DefeatReason()
	if defeat == scenario.DefeatNone {
		// If we exit without an explicit reason, treat it as disconnected.
		defeat = scenario.DefeatDisconnected
	}
	s.stats.SurvivedMs = endedAt.Sub(s.startedAt).Milliseconds()

	s.recordEvent(EventSessionEnded, map[string]any{
		"survived_ms":        s.stats.SurvivedMs,
		"final_phase":        finalPhase,
		"defeat_reason":      defeat,
		"shots_fired":        s.stats.ShotsFired,
		"shots_hit":          s.stats.ShotsHit,
		"damage_taken":       s.stats.DamageTaken,
		"troops_neutralized": s.stats.TroopsNeutralized,
	})

	s.sendNonBlocking(protocol.ServerMessage{
		Type: protocol.ServerMsgSessionEnded,
		Payload: protocol.SessionEndedPayload{
			SessionID:         s.id,
			SurvivedMs:        s.stats.SurvivedMs,
			FinalPhase:        finalPhase,
			DefeatReason:      defeat,
			ShotsFired:        s.stats.ShotsFired,
			ShotsHit:          s.stats.ShotsHit,
			DamageTaken:       s.stats.DamageTaken,
			TroopsNeutralized: s.stats.TroopsNeutralized,
			EventsRecorded:    s.stats.EventsRecorded,
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := s.sessionRepo.End(ctx, storage.SessionEndRecord{
		ID:                s.id,
		EndedAt:           endedAt.UTC(),
		SurvivedMs:        s.stats.SurvivedMs,
		FinalPhase:        string(finalPhase),
		DefeatReason:      string(defeat),
		ShotsFired:        s.stats.ShotsFired,
		ShotsHit:          s.stats.ShotsHit,
		DamageTaken:       s.stats.DamageTaken,
		TroopsNeutralized: s.stats.TroopsNeutralized,
	}); err != nil {
		s.log.Warn("session end persist failed", "err", err)
	}
}

func playerToSnapshot(p *state.CivilianPlayerState) protocol.PlayerSnapshot {
	return protocol.PlayerSnapshot{
		ID:                    p.ID,
		Name:                  p.Name,
		Position:              p.Position,
		Yaw:                   p.Yaw,
		Pitch:                 p.Pitch,
		HP:                    p.HP,
		MaxHP:                 p.MaxHP,
		Ammo:                  p.Ammo,
		MaxAmmo:               p.MaxAmmo,
		IsAlive:               p.IsAlive,
		LastProcessedInputSeq: p.LastProcessedInputSeq,
		SurvivalTimeMs:        p.SurvivalTimeMs,
		Morale:                p.Morale,
	}
}

func troopToSnapshot(t *state.MartialTroopState) protocol.TroopSnapshot {
	return protocol.TroopSnapshot{
		ID:       t.ID,
		Position: t.Position,
		Yaw:      t.Yaw,
		HP:       t.HP,
		MaxHP:    t.MaxHP,
		State:    string(t.State),
		IsAlive:  t.IsAlive,
		SquadID:  t.SquadID,
	}
}

func clonePoint(v gmath.Vector3) *gmath.Vector3 {
	out := v
	return &out
}
