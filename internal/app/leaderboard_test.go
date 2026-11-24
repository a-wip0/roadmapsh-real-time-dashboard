package app

import (
	"context"
	"testing"
)

func TestSubmitRedis(t *testing.T) {
	a := newTestApp(t)
	ctx := context.Background()

	for _, c := range []struct {
		user, game string
		score      int64
		improved   bool
		best       int64
	}{
		{"alice", "chess", 100, true, 100},
		{"alice", "chess", 50, false, 100}, // lower score keeps the best
		{"alice", "pong", 30, true, 30},
		{"bob", "chess", 120, true, 120},
		{"alice", "chess", 110, true, 110}, // global total must move by +10 only
	} {
		improved, best, err := a.submitRedis(ctx, c.user, c.game, c.score)
		if err != nil || improved != c.improved || best != c.best {
			t.Fatalf("%+v -> improved=%v best=%d err=%v", c, improved, best, err)
		}
	}

	g, err := a.top(ctx, globalKey, 10)
	if err != nil || len(g) != 2 || g[0] != (Entry{1, "alice", 140}) || g[1] != (Entry{2, "bob", 120}) {
		t.Errorf("global = %+v err=%v", g, err)
	}
	e, ok, err := a.rank(ctx, gameKey("chess"), "alice")
	if err != nil || !ok || e.Rank != 2 || e.Score != 110 {
		t.Errorf("alice chess = %+v ok=%v err=%v", e, ok, err)
	}
	if _, ok, _ := a.rank(ctx, gameKey("chess"), "nobody"); ok {
		t.Error("unknown user should be unranked")
	}
}

func TestParseLimit(t *testing.T) {
	for in, want := range map[string]int64{"": 10, "abc": 10, "0": 10, "5": 5, "1000": 100} {
		if got := parseLimit(in, 10, 100); got != want {
			t.Errorf("parseLimit(%q) = %d, want %d", in, got, want)
		}
	}
}
