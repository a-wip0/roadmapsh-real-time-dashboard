package app

import (
	"bufio"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestStream(t *testing.T) {
	ts := newTestServer(t)
	alice := signup(t, ts.URL, "alice")

	resp, err := http.Get(ts.URL + "/leaderboard/stream")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	events := make(chan string, 4)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			if d, ok := strings.CutPrefix(sc.Text(), "data: "); ok {
				events <- d
			}
		}
	}()
	next := func() string {
		t.Helper()
		select {
		case e := <-events:
			return e
		case <-time.After(3 * time.Second):
			t.Fatal("timed out waiting for SSE event")
			return ""
		}
	}
	if e := next(); e != "[]" {
		t.Errorf("initial snapshot = %s", e)
	}
	call(t, "POST", ts.URL+"/scores", alice, map[string]any{"game": "chess", "score": 42}, nil)
	if e := next(); !strings.Contains(e, `"alice"`) || !strings.Contains(e, "42") {
		t.Errorf("update after new best = %s", e)
	}
	// a lower score is not a new personal best, so nothing is pushed
	call(t, "POST", ts.URL+"/scores", alice, map[string]any{"game": "chess", "score": 10}, nil)
	select {
	case e := <-events:
		t.Errorf("unexpected event: %s", e)
	case <-time.After(300 * time.Millisecond):
	}
}
