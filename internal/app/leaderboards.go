package app

import "net/http"

func (a *App) globalBoard(w http.ResponseWriter, r *http.Request) {
	a.board(w, r, globalKey)
}

func (a *App) gameBoard(w http.ResponseWriter, r *http.Request) {
	a.board(w, r, gameKey(r.PathValue("game")))
}

func (a *App) board(w http.ResponseWriter, r *http.Request, key string) {
	es, err := a.top(r.Context(), key, parseLimit(r.URL.Query().Get("limit"), 10, 100))
	if err != nil {
		errJSON(w, 502, "leaderboard unavailable")
		return
	}
	writeJSON(w, 200, es)
}

func (a *App) myRank(w http.ResponseWriter, r *http.Request, user string) {
	key := globalKey
	if g := r.URL.Query().Get("game"); g != "" {
		key = gameKey(g)
	}
	e, ok, err := a.rank(r.Context(), key, user)
	if err != nil {
		errJSON(w, 502, "leaderboard unavailable")
		return
	}
	if !ok {
		errJSON(w, 404, "no score submitted yet")
		return
	}
	writeJSON(w, 200, e)
}
