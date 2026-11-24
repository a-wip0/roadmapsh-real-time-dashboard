package app

import (
	"net/http"
	"regexp"
	"time"
)

var gameRe = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,64}$`)

func (a *App) submit(w http.ResponseWriter, r *http.Request, user string) {
	var req struct {
		Game  string `json:"game"`
		Score int64  `json:"score"`
	}
	if !decode(w, r, &req) {
		return
	}
	if !gameRe.MatchString(req.Game) || req.Score < 0 || req.Score > 1<<40 {
		errJSON(w, 400, "game must match [A-Za-z0-9_.-]{1,64}; score must be 0..2^40")
		return
	}
	if _, err := a.db.Exec(`INSERT INTO scores(username, game, score, created_at) VALUES(?,?,?,?)`,
		user, req.Game, req.Score, time.Now().Unix()); err != nil {
		errJSON(w, 500, "internal error")
		return
	}
	improved, best, err := a.submitRedis(r.Context(), user, req.Game, req.Score)
	if err != nil {
		errJSON(w, 502, "leaderboard unavailable")
		return
	}
	g, _, _ := a.rank(r.Context(), gameKey(req.Game), user)
	gl, _, _ := a.rank(r.Context(), globalKey, user)
	writeJSON(w, 201, map[string]any{
		"game": req.Game, "score": req.Score, "personal_best": best, "new_personal_best": improved,
		"game_rank": g.Rank, "global_rank": gl.Rank,
	})
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
