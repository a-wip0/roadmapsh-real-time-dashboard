package app

import (
	"context"
	"strconv"
)

const globalKey = "lb:global"

func gameKey(game string) string { return "lb:game:" + game }

// submitLua keeps each user's best score per game in lb:game:<game> and the
// sum of those bests across games in lb:global, atomically.
// Returns {improved(0/1), newBest}.
const submitLua = `
local old = redis.call('ZSCORE', KEYS[1], ARGV[2])
local s = tonumber(ARGV[1])
if (not old) or s > tonumber(old) then
  redis.call('ZADD', KEYS[1], s, ARGV[2])
  redis.call('ZINCRBY', KEYS[2], s - (old and tonumber(old) or 0), ARGV[2])
  return {1, s}
end
return {0, tonumber(old)}`

type Entry struct {
	Rank     int64  `json:"rank"`
	Username string `json:"username"`
	Score    int64  `json:"score"`
}

// Submit records the score in Redis and reports whether it is a new personal best.
func (a *App) submitRedis(ctx context.Context, user, game string, score int64) (bool, int64, error) {
	res, err := a.score.Run(ctx, a.rdb, []string{gameKey(game), globalKey}, score, user).Int64Slice()
	if err != nil {
		return false, 0, err
	}
	return res[0] == 1, res[1], nil
}

func (a *App) top(ctx context.Context, key string, limit int64) ([]Entry, error) {
	zs, err := a.rdb.ZRevRangeWithScores(ctx, key, 0, limit-1).Result()
	if err != nil {
		return nil, err
	}
	out := make([]Entry, len(zs))
	for i, z := range zs {
		out[i] = Entry{Rank: int64(i + 1), Username: z.Member.(string), Score: int64(z.Score)}
	}
	return out, nil
}

// rank returns the user's 1-based rank and score; ok=false if unranked.
func (a *App) rank(ctx context.Context, key, user string) (Entry, bool, error) {
	r, err := a.rdb.ZRevRank(ctx, key, user).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			return Entry{}, false, nil
		}
		return Entry{}, false, err
	}
	s, err := a.rdb.ZScore(ctx, key, user).Result()
	if err != nil {
		return Entry{}, false, err
	}
	return Entry{Rank: r + 1, Username: user, Score: int64(s)}, true, nil
}

func parseLimit(s string, def, max int64) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n < 1 {
		return def
	}
	if n > max {
		return max
	}
	return n
}
