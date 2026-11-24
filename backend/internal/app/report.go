package app

import (
	"errors"
	"net/http"
	"time"
)

func parseTime(s string, endOfDay bool) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return t, errors.New("use RFC3339 or YYYY-MM-DD")
	}
	if endOfDay {
		t = t.Add(24*time.Hour - time.Second)
	}
	return t, nil
}

// report: GET /reports/top-players?period=day|week|month  OR  ?from=&to=
// optional &game= and &limit=. Each player's total is the sum of their best
// score per game within the period.
func (a *App) report(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	now := time.Now().UTC()
	from, to := time.Unix(0, 0), now
	switch q.Get("period") {
	case "":
	case "day":
		from = now.Add(-24 * time.Hour)
	case "week":
		from = now.AddDate(0, 0, -7)
	case "month":
		from = now.AddDate(0, -1, 0)
	default:
		errJSON(w, 400, "period must be day, week or month")
		return
	}
	var err error
	if v := q.Get("from"); v != "" {
		if from, err = parseTime(v, false); err != nil {
			errJSON(w, 400, "from: "+err.Error())
			return
		}
	}
	if v := q.Get("to"); v != "" {
		if to, err = parseTime(v, true); err != nil {
			errJSON(w, 400, "to: "+err.Error())
			return
		}
	}
	sub := `SELECT username, game, MAX(score) AS best FROM scores WHERE created_at BETWEEN ? AND ?`
	args := []any{from.Unix(), to.Unix()}
	if g := q.Get("game"); g != "" {
		sub += ` AND game=?`
		args = append(args, g)
	}
	sub += ` GROUP BY username, game`
	limit := parseLimit(q.Get("limit"), 10, 100)
	rows, err := a.db.Query(`SELECT username, SUM(best) t FROM (`+sub+`) GROUP BY username ORDER BY t DESC, username LIMIT ?`,
		append(args, limit)...)
	if err != nil {
		errJSON(w, 500, "internal error")
		return
	}
	defer rows.Close()
	out := []Entry{}
	for rows.Next() {
		var e Entry
		if rows.Scan(&e.Username, &e.Score) == nil {
			e.Rank = int64(len(out) + 1)
			out = append(out, e)
		}
	}
	writeJSON(w, 200, map[string]any{"from": from.UTC(), "to": to.UTC(), "game": q.Get("game"), "players": out})
}
