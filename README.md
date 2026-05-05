# mayday-server

**Mayday**(싱글플레이 웹 3D FPS)의 권위형(authoritative) Go 게임 서버. 클라이언트는 입력 수집과 렌더링만 담당하고, 시뮬레이션 상태·히트 검증·AI·시나리오 진행·세션 이벤트 로그는 모두 서버가 소유한다.


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

### 동시성 모델

- 연결당 **Session goroutine** 하나. 시뮬레이션 상태를 변경하는 유일한 고루틴이다.
- 연결당 **WebSocket reader** + **WebSocket writer** 각 하나, 둘 다 버퍼드 채널 사용.
- 세션마다 **이벤트 퍼시스터** 고루틴 하나. 버퍼드 채널을 PostgreSQL로 비동기 드레인하므로 DB가 느려도 틱 루프는 멈추지 않는다.
- 뮤텍스는 세션 맵과 연결별 송신 버퍼 주변에만 등장한다.

### 패키지 구조

```
cmd/server/                  main()
internal/config/             환경변수 기반 타입드 컨피그
internal/logger/             slog 설정
internal/observability/      카운터, uptime
internal/protocol/           엔벨로프 + 타입드 클라이언트/서버 메시지
internal/transport/http/     /health, HTTP 부트스트랩
internal/transport/websocket/  gorilla/websocket reader & writer
internal/storage/            pgx 풀 + Event/Session repo (+ noop / memory 폴백)
internal/game/               Session, SessionManager, 틱 루프, 이벤트
internal/game/state/         CivilianPlayerState, MartialTroopState
internal/game/math/          Vector3, raycast
internal/game/scenario/      Phase, DefeatReason, ScenarioDirector
internal/game/systems/       movement, shooting, damage, defeat, objective
internal/ai/                 FSM 상태, 지각, 순수 Decide()
internal/ai/behavior/        상태별 행동 헬퍼
migrations/                  goose 기반 SQL
tests/                       크로스 패키지 테스트
```

### 시나리오 페이즈

`INITIAL_CONTACT → ESCALATION → REINFORCEMENT → ENCIRCLEMENT → FINAL_STAND → DEFEAT`. `VICTORY`는 존재하지 않는다. 패배 사유: `PLAYER_KILLED`, `OVERRUN`, `AMMO_EXHAUSTED`, `ENCIRCLED`, `SCRIPTED_FINAL_STAND`, `DISCONNECTED`.

### 진압군 AI 상태

`PATROL`, `ADVANCE`, `CHASE`, `ATTACK`, `SUPPRESS`, `FLANK`, `BLOCK_EXIT`, `CALL_REINFORCEMENT`, `TAKE_COVER`, `DEAD`. 결정 로직은 순수 함수 `ai.Decide(input) → (state, []Action)`.

---

## API 스펙

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

서브프로토콜 없음. 모든 프레임은 UTF-8 JSON이며, 다음 엔벨로프로 감싼다:

```json
{ "type": "<message_type>", "payload": { ... } }
```

클라이언트의 첫 메시지는 **반드시** `start_session`이어야 한다. 그 외에는 `error` 프레임(`code: session_not_started`)이 응답된다.

---

### Client → Server 메시지

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

`delta_ms`는 텔레포트 방지를 위해 서버에서 100ms로 클램프된다.

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

레이캐스트는 서버가 직접 수행한다. 히트 결과는 전적으로 서버 계산이며, 클라이언트가 보낸 히트 정보는 무시된다.

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

### Server → Client 메시지

#### `welcome`

```json
{ "type": "welcome", "payload": { "server_version": "mayday-mvp", "server_time": 1714824000000 } }
```

#### `session_started`

```json
{ "type": "session_started", "payload": { "session_id": "uuid", "tick_rate": 30, "started_at": 1714824000000 } }
```

#### `state_snapshot` (스냅샷 틱마다, 기본 15Hz)

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

`reason` 값: `hit`, `miss`, `dead`, `no_ammo`, `fire_rate`, `bad_direction`, `no_player`.

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

압력값이 0.05 이상 변할 때만 송신된다.

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

알림 용도의 경량 메시지. 풀 페이로드는 PostgreSQL의 `game_events`에 저장된다.

#### `pong`

```json
{ "type": "pong", "payload": { "client_time": 123456789, "server_time": 1714824000000 } }
```

#### `error`

```json
{ "type": "error", "payload": { "code": "parse_error", "message": "..." } }
```

에러 코드: `parse_error`, `session_not_started`, 그리고 파서 단계 에러인 `invalid_json`, `unknown_message_type`, `malformed_payload`, `empty_message`.

---

## 영속화 스키마

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

인덱스: `game_events(session_id, server_tick)`, `game_events(type)`, `game_sessions(started_at)`.

이벤트 타입: `SESSION_STARTED`, `PHASE_CHANGED`, `PRESSURE_CHANGED`, `TROOP_SPAWNED`, `PLAYER_SHOT`, `PLAYER_HIT_TROOP`, `PLAYER_DAMAGED`, `PLAYER_DIED`, `DEFEAT_TRIGGERED`, `SESSION_ENDED`.

---

## 실행

```bash
cp .env.example .env
make run                  # 로컬 실행
make test                 # go test -count=1 -race ./...
docker compose up --build # postgres + 서버 동시 기동
make migrate-up           # goose 필요
```

`DATABASE_URL`이 비어있거나 접속 불가일 경우 서버는 noop 리포지토리로 부팅된다 — 시뮬레이션은 정상 동작한다.

기본 포트: `:3001`.

## 환경변수

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
