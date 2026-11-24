package app

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"time"
)

var gameRe = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,64}$`)

var (
	// ErrInvalidInput means the game name or score failed validation.
	ErrInvalidInput = errors.New("game must match [A-Za-z0-9_.-]{1,64}; score must be 0..2^40")
	// ErrUnavailable means the Redis leaderboard could not be updated.
	ErrUnavailable = errors.New("leaderboard unavailable")
)

// SubmitResult is what a player learns after submitting a score.
type SubmitResult struct {
	Game            string `json:"game"`
	Score           int64  `json:"score"`
	PersonalBest    int64  `json:"personal_best"`
	NewPersonalBest bool   `json:"new_personal_best"`
	GameRank        int64  `json:"game_rank"`
	GlobalRank      int64  `json:"global_rank"`
}

// Submit records a score for user: it is appended to the SQLite history, then
// the Redis boards are updated and, on a new personal best, SSE clients are
// notified. It is the single write path shared by the HTTP and gRPC APIs.
// The caller is responsible for knowing that user exists.
func (a *App) Submit(ctx context.Context, user, game string, score int64) (*SubmitResult, error) {
	if !gameRe.MatchString(game) || score < 0 || score > 1<<40 {
		return nil, ErrInvalidInput
	}
	if _, err := a.db.ExecContext(ctx, `INSERT INTO scores(username, game, score, created_at) VALUES(?,?,?,?)`,
		user, game, score, time.Now().Unix()); err != nil {
		return nil, err
	}
	improved, best, err := a.submitRedis(ctx, user, game, score)
	if err != nil {
		return nil, ErrUnavailable
	}
	if improved {
		a.broadcast()
	}
	g, _, _ := a.rank(ctx, gameKey(game), user)
	gl, _, _ := a.rank(ctx, globalKey, user)
	return &SubmitResult{Game: game, Score: score, PersonalBest: best, NewPersonalBest: improved,
		GameRank: g.Rank, GlobalRank: gl.Rank}, nil
}

func (a *App) submit(w http.ResponseWriter, r *http.Request, user string) {
	var req struct {
		Game  string `json:"game"`
		Score int64  `json:"score"`
	}
	if !decode(w, r, &req) {
		return
	}
	res, err := a.Submit(r.Context(), user, req.Game, req.Score)
	switch {
	case errors.Is(err, ErrInvalidInput):
		errJSON(w, 400, err.Error())
	case errors.Is(err, ErrUnavailable):
		errJSON(w, 502, err.Error())
	case err != nil:
		errJSON(w, 500, "internal error")
	default:
		writeJSON(w, 201, res)
	}
}

func (a *App) history(w http.ResponseWriter, r *http.Request, user string) {
	q := `SELECT game, score, created_at FROM scores WHERE username=?`
	args := []any{user}
	if g := r.URL.Query().Get("game"); g != "" {
		q += ` AND game=?`
		args = append(args, g)
	}
	q += ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, parseLimit(r.URL.Query().Get("limit"), 50, 500))
	rows, err := a.db.Query(q, args...)
	if err != nil {
		errJSON(w, 500, "internal error")
		return
	}
	defer rows.Close()
	type item struct {
		Game      string    `json:"game"`
		Score     int64     `json:"score"`
		CreatedAt time.Time `json:"created_at"`
	}
	out := []item{}
	for rows.Next() {
		var it item
		var ts int64
		if rows.Scan(&it.Game, &it.Score, &ts) == nil {
			it.CreatedAt = time.Unix(ts, 0).UTC()
			out = append(out, it)
		}
	}
	writeJSON(w, 200, out)
}
