package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func (a *App) broadcast() {
	es, err := a.top(context.Background(), globalKey, 10)
	if err != nil {
		return
	}
	b, _ := json.Marshal(es)
	a.hub.publish(b)
}

func (a *App) stream(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		errJSON(w, 500, "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	ch := a.hub.subscribe()
	defer a.hub.unsubscribe(ch)
	// initial snapshot
	if es, err := a.top(r.Context(), globalKey, 10); err == nil {
		b, _ := json.Marshal(es)
		fmt.Fprintf(w, "event: leaderboard\ndata: %s\n\n", b)
	}
	fl.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case b := <-ch:
			fmt.Fprintf(w, "event: leaderboard\ndata: %s\n\n", b)
			fl.Flush()
		}
	}
}
