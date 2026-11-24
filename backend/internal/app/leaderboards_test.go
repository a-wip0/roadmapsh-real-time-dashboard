package app

import "testing"

func TestLeaderboards(t *testing.T) {
	b := newTestServer(t).URL
	alice := signup(t, b, "alice")
	bob := signup(t, b, "bob")
	post := func(token, game string, score int) {
		t.Helper()
		if code := call(t, "POST", b+"/scores", token, map[string]any{"game": game, "score": score}, nil); code != 201 {
			t.Fatalf("submit = %d", code)
		}
	}
	post(alice, "chess", 100)
	post(alice, "chess", 50)
	post(alice, "pong", 30)
	post(bob, "chess", 120)

	// global = sum of each user's best per game: alice 130, bob 120
	var global []Entry
	call(t, "GET", b+"/leaderboard", "", nil, &global)
	if len(global) != 2 || global[0] != (Entry{1, "alice", 130}) || global[1] != (Entry{2, "bob", 120}) {
		t.Errorf("global = %+v", global)
	}
	call(t, "GET", b+"/leaderboard?limit=1", "", nil, &global)
	if len(global) != 1 {
		t.Errorf("limit=1 returned %d", len(global))
	}

	var chess []Entry
	call(t, "GET", b+"/leaderboard/chess", "", nil, &chess)
	if len(chess) != 2 || chess[0].Username != "bob" || chess[1].Score != 100 {
		t.Errorf("chess = %+v", chess)
	}
	call(t, "GET", b+"/leaderboard/unknown-game", "", nil, &chess)
	if len(chess) != 0 {
		t.Errorf("unknown game = %+v", chess)
	}

	var me Entry
	if code := call(t, "GET", b+"/rankings/me?game=chess", alice, nil, &me); code != 200 || me.Rank != 2 || me.Score != 100 {
		t.Errorf("alice chess = %d %+v", code, me)
	}
	if code := call(t, "GET", b+"/rankings/me", alice, nil, &me); code != 200 || me.Rank != 1 || me.Score != 130 {
		t.Errorf("alice global = %d %+v", code, me)
	}
	if code := call(t, "GET", b+"/rankings/me?game=nope", alice, nil, nil); code != 404 {
		t.Errorf("unranked = %d", code)
	}
	if code := call(t, "GET", b+"/rankings/me", "", nil, nil); code != 401 {
		t.Errorf("unauthenticated = %d", code)
	}
}
