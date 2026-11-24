# Real-time Leaderboard System

Solution for the [roadmap.sh Realtime Leaderboard System](https://roadmap.sh/projects/realtime-leaderboard-system) project.
Go + Redis sorted sets (rankings) + SQLite (users and score history) + JWT auth.

## Run

```bash
redis-server --daemonize yes
cd backend
JWT_SECRET=change-me go run ./cmd/server      # listens on :8080
go test ./...                                 # needs Redis on localhost:6379 (uses DB 15, flushes it)
```

Env vars: `ADDR` (`:8080`), `REDIS_URL` (`redis://localhost:6379/0`), `DB_PATH` (`leaderboard.db`), `JWT_SECRET`, `TEST_REDIS_URL`.

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

## Design

- `lb:game:<game>` sorted set holds each user's best score; `lb:global` holds the sum of those bests.
  A Lua script updates both atomically, so concurrent submissions can't corrupt the global total.
- Every submission is also written to SQLite, which backs score history and period reports.
- SSE fan-out is in-process; with multiple instances, swap it for Redis pub/sub.
