# mayday-server

Authoritative Go game server for **Mayday**, a single-player web 3D FPS. The client only collects input and renders; the server owns all simulation state, hit validation, AI, scenario progression, and the session event log.


```mermaid
flowchart LR
    Client[Browser Client]
    GW[/WebSocket Gateway/]
    Mgr[SessionManager]
    Sess[Session goroutine]
    Tick[Tick Loop 30Hz]
    Sys[Game Systems]
    AI[Troop AI - FSM]
    Dir[ScenarioDirector]
    State[Authoritative State]
    Snap[Snapshot 15Hz]
    DB[(PostgreSQL)]

    Client <-- JSON over WS --> GW
    GW --> Mgr --> Sess
    Sess --> Tick
    Tick --> Sys
    Tick --> AI
    Tick --> Dir
    Tick --> State
    Tick --> Snap --> Client
    Sess -- async event buffer --> DB
```

### Concurrency model

- One **Session goroutine** per connection. It is the only goroutine that mutates simulation state.
- One **WebSocket reader** + **WebSocket writer** per connection, both buffered.
- One **event persister** goroutine per session, drains a buffered channel into PostgreSQL so a slow DB never stalls the tick loop.
- Mutexes appear only around the session map and the per-connection send buffer.

### Package layout

```
cmd/server/                  main()
internal/config/             env-backed typed config
internal/logger/             slog setup
internal/observability/      counters, uptime
internal/protocol/           envelope + typed client/server messages
internal/transport/http/     /health, HTTP bootstrap
internal/transport/websocket/  gorilla/websocket reader & writer
internal/storage/            pgx pool + Event/Session repos (+ noop / memory)
internal/game/               Session, SessionManager, tick loop, events
internal/game/state/         CivilianPlayerState, MartialTroopState
internal/game/math/          Vector3, raycast
internal/game/scenario/      Phase, DefeatReason, ScenarioDirector
internal/game/systems/       movement, shooting, damage, defeat, objective
internal/ai/                 FSM states, perception, pure Decide()
internal/ai/behavior/        per-state behavior helpers
migrations/                  goose-driven SQL
tests/                       cross-package tests
```

### Scenario phases

`INITIAL_CONTACT → ESCALATION → REINFORCEMENT → ENCIRCLEMENT → FINAL_STAND → DEFEAT`. There is no `VICTORY`. Defeat reasons: `PLAYER_KILLED`, `OVERRUN`, `AMMO_EXHAUSTED`, `ENCIRCLED`, `SCRIPTED_FINAL_STAND`, `DISCONNECTED`.

### Troop AI states

`PATROL`, `ADVANCE`, `CHASE`, `ATTACK`, `SUPPRESS`, `FLANK`, `BLOCK_EXIT`, `CALL_REINFORCEMENT`, `TAKE_COVER`, `DEAD`. Pure `ai.Decide(input) → (state, []Action)`.

---

## API spec

### HTTP

#### `GET /health`

```json
{
  "status": "ok",
  "service": "mayday-server",
  "uptime": 123,
  "timestamp": "2026-05-04T12:34:56Z"
}
```

### WebSocket

#### `GET /ws`

Subprotocol: none. All frames are UTF-8 JSON. Every frame is wrapped in:

```json
{ "type": "<message_type>", "payload": { ... } }
```

The first message from the client **must** be `start_session`. Anything else returns an `error` frame with code `session_not_started`.

---

### Client → Server messages

#### `start_session`

```json
{ "type": "start_session", "payload": { "player_name": "anonymous" } }
```

#### `player_input`

```json
{
  "type": "player_input",
  "payload": {
    "seq": 12,
    "move": { "forward": true, "backward": false, "left": false, "right": true },
    "delta_ms": 16
  }
}
```

`delta_ms` is clamped server-side (max 100 ms) to prevent teleport.

#### `player_look`

```json
{ "type": "player_look", "payload": { "yaw": 1.5, "pitch": -0.2 } }
```

#### `shoot`

```json
{
  "type": "shoot",
  "payload": {
    "seq": 18,
    "origin":    { "x": 0, "y": 1.6, "z": 0 },
    "direction": { "x": 0, "y": 0,   "z": 1 },
    "client_time": 123456789
  }
}
```

The server runs its own raycast. The reported hit (if any) is computed server-side; client-supplied hit results are ignored.

#### `reload`

```json
{ "type": "reload", "payload": {} }
```

#### `interact`

```json
{ "type": "interact", "payload": { "target_id": "object-id" } }
```

#### `ping`

```json
{ "type": "ping", "payload": { "client_time": 123456789 } }
```

---

### Server → Client messages

#### `welcome`

```json
{ "type": "welcome", "payload": { "server_version": "mayday-mvp", "server_time": 1714824000000 } }
```

#### `session_started`

```json
{ "type": "session_started", "payload": { "session_id": "uuid", "tick_rate": 30, "started_at": 1714824000000 } }
```

#### `state_snapshot` (every snapshot tick, default 15Hz)

```json
{
  "type": "state_snapshot",
  "payload": {
    "server_tick": 1024,
    "session_id": "uuid",
    "scenario_phase": "INITIAL_CONTACT",
    "pressure_level": 0.35,
    "encirclement_level": 0.20,
    "player": {
      "id": "uuid", "name": "jin",
      "position": { "x": 0, "y": 1.6, "z": 0 },
      "yaw": 0, "pitch": 0,
      "hp": 100, "max_hp": 100,
      "ammo": 24, "max_ammo": 24,
      "is_alive": true,
      "last_processed_input_seq": 12,
      "survival_time_ms": 5400,
      "morale": 1.0
    },
    "troops": [
      {
        "id": "uuid",
        "position": { "x": 12, "y": 0, "z": 8 },
        "yaw": 1.2,
        "hp": 60, "max_hp": 60,
        "state": "CHASE",
        "is_alive": true,
        "squad_id": "alpha"
      }
    ],
    "events": [
      { "type": "TROOP_SPAWNED", "server_tick": 1020 }
    ]
  }
}
```

#### `troop_spawned`

```json
{ "type": "troop_spawned", "payload": { "troop": { /* TroopSnapshot */ }, "server_tick": 1024 } }
```

#### `shot_result`

```json
{
  "type": "shot_result",
  "payload": {
    "seq": 18,
    "accepted": true,
    "reason": "hit",
    "hit_troop_id": "uuid",
    "hit_distance": 8.4,
    "damage_dealt": 25,
    "troop_killed": false,
    "ammo_left": 23
  }
}
```

`reason` is one of: `hit`, `miss`, `dead`, `no_ammo`, `fire_rate`, `bad_direction`, `no_player`.

#### `damage_taken`

```json
{ "type": "damage_taken", "payload": { "source": "martial_troop", "source_id": "uuid", "damage": 8, "remaining_hp": 92 } }
```

#### `player_died`

```json
{ "type": "player_died", "payload": { "session_id": "uuid", "tick": 1500 } }
```

#### `scenario_phase_changed`

```json
{ "type": "scenario_phase_changed", "payload": { "previous_phase": "ESCALATION", "current_phase": "REINFORCEMENT", "tick": 1800 } }
```

#### `pressure_changed`

```json
{ "type": "pressure_changed", "payload": { "pressure_level": 0.62, "encirclement_level": 0.45 } }
```

Emitted only when pressure moves by ≥ 0.05.

#### `defeat_triggered`

```json
{ "type": "defeat_triggered", "payload": { "reason": "SCRIPTED_FINAL_STAND", "tick": 12600 } }
```

#### `session_ended`

```json
{
  "type": "session_ended",
  "payload": {
    "session_id": "uuid",
    "survived_ms": 420000,
    "final_phase": "DEFEAT",
    "defeat_reason": "SCRIPTED_FINAL_STAND",
    "shots_fired": 18,
    "shots_hit": 11,
    "damage_taken": 64,
    "troops_neutralized": 7,
    "events_recorded": 152
  }
}
```

#### `event_logged`

```json
{ "type": "event_logged", "payload": { "type": "PLAYER_HIT_TROOP", "server_tick": 950 } }
```

Lightweight notification only. Full payloads live in `game_events` (PostgreSQL).

#### `pong`

```json
{ "type": "pong", "payload": { "client_time": 123456789, "server_time": 1714824000000 } }
```

#### `error`

```json
{ "type": "error", "payload": { "code": "parse_error", "message": "..." } }
```

Error codes: `parse_error`, `session_not_started`, plus the parse error variants `invalid_json`, `unknown_message_type`, `malformed_payload`, `empty_message`.

---

## Persisted schema

```sql
game_sessions(
  id UUID PK, player_name TEXT, started_at TIMESTAMPTZ, ended_at TIMESTAMPTZ,
  survived_ms BIGINT, final_phase TEXT, defeat_reason TEXT,
  shots_fired INT, shots_hit INT, damage_taken INT, troops_neutralized INT,
  created_at TIMESTAMPTZ
)

game_events(
  id UUID PK, session_id UUID FK,
  type TEXT, server_tick BIGINT, payload JSONB, created_at TIMESTAMPTZ
)
```

Indexes: `game_events(session_id, server_tick)`, `game_events(type)`, `game_sessions(started_at)`.

Event types: `SESSION_STARTED`, `PHASE_CHANGED`, `PRESSURE_CHANGED`, `TROOP_SPAWNED`, `PLAYER_SHOT`, `PLAYER_HIT_TROOP`, `PLAYER_DAMAGED`, `PLAYER_DIED`, `DEFEAT_TRIGGERED`, `SESSION_ENDED`.

---

## Run

```bash
cp .env.example .env
make run                  # local
make test                 # go test -count=1 -race ./...
docker compose up --build # postgres + server
make migrate-up           # requires goose
```

If `DATABASE_URL` is unset or unreachable, the server boots with no-op repositories — the simulation still runs.

Default port: `:3001`.

## Config (env)

| Var | Default |
|---|---|
| `PORT` | `3001` |
| `DATABASE_URL` | `postgres://mayday:mayday@localhost:5432/mayday?sslmode=disable` |
| `TICK_RATE` / `SNAPSHOT_RATE` | `30` / `15` |
| `INITIAL_TROOP_COUNT` / `MAX_TROOP_COUNT` | `4` / `30` |
| `TROOP_DETECTION_RANGE` / `TROOP_ATTACK_RANGE` / `TROOP_DAMAGE` / `TROOP_MOVE_SPEED` | `35` / `22` / `8` / `4` |
| `PLAYER_MAX_HP` / `PLAYER_MAX_AMMO` / `PLAYER_MOVE_SPEED` / `PLAYER_SHOOT_DAMAGE` | `100` / `24` / `6` / `25` |
| `SHOOT_MAX_DISTANCE` / `SHOOT_ANGLE_THRESHOLD` / `FIRE_RATE_LIMIT_MS` | `60` / `0.96` / `250` |
| `SESSION_MAX_DURATION_MS` / `FINAL_STAND_AFTER_MS` / `FORCE_DEFEAT_AFTER_MS` | `600000` / `300000` / `420000` |
| `SESSION_EVENT_BUFFER_SIZE` / `CLIENT_SEND_BUFFER_SIZE` | `512` / `64` |
