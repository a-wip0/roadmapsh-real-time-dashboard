package app

import (
	"encoding/json"
	"net/http"
)

func (a *App) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("POST /auth/register", a.register)
	m.HandleFunc("POST /auth/login", a.login)
	m.Handle("POST /scores", a.auth(a.submit))
	m.Handle("GET /scores/history", a.auth(a.history))
	m.HandleFunc("GET /leaderboard", a.globalBoard)
	m.HandleFunc("GET /leaderboard/{game}", a.gameBoard)
	m.Handle("GET /rankings/me", a.auth(a.myRank))
	return m
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func errJSON(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<16)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		errJSON(w, 400, "invalid JSON body")
		return false
	}
	return true
}
