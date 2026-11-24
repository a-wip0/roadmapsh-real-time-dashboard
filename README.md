# Real-time Leaderboard System

Solution for the [roadmap.sh Realtime Leaderboard System](https://roadmap.sh/projects/realtime-leaderboard-system) project.
Go + Redis sorted sets (rankings) + SQLite (users and score history) + JWT auth, with a React + TypeScript front end.

```
backend/    Go API (net/http + gRPC, Redis, SQLite)
frontend/   Vite + React + TypeScript UI
simulator/  Python load generator that feeds scores over gRPC
proto/      gRPC contract shared by backend and simulator
```

## Quick start (everything, locally)

```bash
docker compose up -d redis            # or: redis-server --daemonize yes

# terminal 1: API (:8080) and gRPC ingest (:9090)
cd backend && INGEST_API_KEY=dev-key JWT_SECRET=change-me go run ./cmd/server

# terminal 2: UI on http://localhost:5173
cd frontend && npm install && npm run dev

# terminal 3: fake traffic, watch the UI update live
python3 -m venv simulator/.venv && simulator/.venv/bin/pip install -r simulator/requirements.txt
simulator/.venv/bin/python simulator/simulate.py --api-key dev-key
```

## Run

```bash
redis-server --daemonize yes
cd backend
JWT_SECRET=change-me go run ./cmd/server      # listens on :8080
go test ./...                                 # needs Redis on localhost:6379 (uses DB 15, flushes it)
```

Env vars: `ADDR` (`:8080`), `GRPC_ADDR` (`:9090`), `INGEST_API_KEY` (enables gRPC ingest; unset = disabled), `REDIS_URL` (`redis://localhost:6379/0`), `DB_PATH` (`leaderboard.db`), `JWT_SECRET`, `TEST_REDIS_URL`.

### Front end

```bash
cd frontend
npm install
npm run dev        # http://localhost:5173, proxies /api/* to the backend on :8080
npm run build      # type-check + production bundle in frontend/dist
```

Set `BACKEND_URL` if the API isn't on `http://localhost:8080`. The UI covers login/register, a live global/per-game
leaderboard (SSE), score submission with rank feedback, your history, and the top-players report.

## API

| Method & path | Auth | Description |
|---|---|---|
| `POST /auth/register` `{username,password}` | – | Create user (password 8–72 chars) |
| `POST /auth/login` `{username,password}` | – | Returns `{token}` (JWT, 24h) |
| `POST /scores` `{game,score}` | Bearer | Submit a score; returns personal best and ranks |
| `GET /scores/history?game=&limit=` | Bearer | Your submitted scores, newest first |
| `GET /leaderboard?limit=` | – | Global board: sum of each user's best score per game |
| `GET /leaderboard/{game}?limit=` | – | Per-game board (best score per user) |
| `GET /leaderboard/stream` | – | Server-Sent Events: top 10 global, pushed on every new personal best |
| `GET /rankings/me?game=` | Bearer | Your rank and score (global, or for one game) |
| `GET /reports/top-players?period=day\|week\|month` or `?from=&to=` (+`&game=&limit=`) | – | Top players for a period, computed from score history |

## gRPC ingest (trusted write path)

For game servers and the simulator. Enabled only when `INGEST_API_KEY` is set. Contract: `proto/leaderboard/v1/leaderboard.proto`.

| RPC | Description |
|---|---|
| `SubmitScore(username, game, score)` | One score. `NOT_FOUND` for an unknown player, `INVALID_ARGUMENT` for bad input, `UNAUTHENTICATED` for a bad key |
| `StreamScores(stream …) → {accepted, rejected}` | Many scores on one connection; bad entries are counted, not fatal |

Callers send the key as `x-api-key` metadata. Players must already exist (register through `POST /auth/register`).
Accepted scores use the same code path as `POST /scores` (`App.Submit`), so Redis, history and SSE behave identically.
The channel is plaintext: keep `:9090` on a private network, or add TLS before exposing it.

```bash
simulator/.venv/bin/python simulator/simulate.py --api-key dev-key --players 50 --rate 20 --games chess,pong,tetris
#   --unary (one call per score)  --duration 30  --seed 1  -v
```

Regenerate bindings after editing the proto: `scripts/gen-proto.sh` (needs `grpcio-tools`, `protoc-gen-go`, `protoc-gen-go-grpc`).

## Design

- `lb:game:<game>` sorted set holds each user's best score; `lb:global` holds the sum of those bests.
  A Lua script updates both atomically, so concurrent submissions can't corrupt the global total.
- Every submission is also written to SQLite, which backs score history and period reports.
- SSE fan-out is in-process; with multiple instances, swap it for Redis pub/sub.
