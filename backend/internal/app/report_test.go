package app

import "testing"

func TestReport(t *testing.T) {
	b := newTestServer(t).URL
	alice := signup(t, b, "alice")
	bob := signup(t, b, "bob")
	for _, s := range []struct {
		token, game string
		score       int
	}{{alice, "chess", 100}, {alice, "chess", 50}, {alice, "pong", 30}, {bob, "chess", 120}} {
		call(t, "POST", b+"/scores", s.token, map[string]any{"game": s.game, "score": s.score}, nil)
	}

	var rep struct{ Players []Entry }
	call(t, "GET", b+"/reports/top-players?period=week", "", nil, &rep)
	if len(rep.Players) != 2 || rep.Players[0] != (Entry{1, "alice", 130}) || rep.Players[1] != (Entry{2, "bob", 120}) {
		t.Errorf("week = %+v", rep.Players)
	}
	call(t, "GET", b+"/reports/top-players?period=day&game=chess", "", nil, &rep)
	if len(rep.Players) != 2 || rep.Players[0].Username != "bob" {
		t.Errorf("chess = %+v", rep.Players)
	}
	call(t, "GET", b+"/reports/top-players?limit=1", "", nil, &rep)
	if len(rep.Players) != 1 {
		t.Errorf("limit = %+v", rep.Players)
	}
	call(t, "GET", b+"/reports/top-players?from=2000-01-01&to=2000-01-02", "", nil, &rep)
	if len(rep.Players) != 0 {
		t.Errorf("past window should be empty: %+v", rep.Players)
	}
	for _, q := range []string{"period=year", "from=nope", "to=2020-13-45"} {
		if code := call(t, "GET", b+"/reports/top-players?"+q, "", nil, nil); code != 400 {
			t.Errorf("%s = %d", q, code)
		}
	}
}
