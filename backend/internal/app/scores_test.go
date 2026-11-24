package app

import "testing"

func TestScores(t *testing.T) {
	b := newTestServer(t).URL
	alice := signup(t, b, "alice")
	bob := signup(t, b, "bob")

	submit := func(token, game string, score int64) map[string]any {
		t.Helper()
		var r map[string]any
		if code := call(t, "POST", b+"/scores", token, map[string]any{"game": game, "score": score}, &r); code != 201 {
			t.Fatalf("submit = %d", code)
		}
		return r
	}
	if r := submit(alice, "chess", 100); r["new_personal_best"] != true || r["game_rank"].(float64) != 1 {
		t.Errorf("first submit = %v", r)
	}
	if r := submit(alice, "chess", 50); r["new_personal_best"] != false || r["personal_best"].(float64) != 100 {
		t.Errorf("lower submit = %v", r)
	}
	submit(alice, "pong", 30)
	if r := submit(bob, "chess", 120); r["game_rank"].(float64) != 1 || r["global_rank"].(float64) != 2 {
		t.Errorf("bob submit = %v", r)
	}

	// validation and auth
	if code := call(t, "POST", b+"/scores", "", map[string]any{"game": "chess", "score": 1}, nil); code != 401 {
		t.Errorf("unauthenticated = %d", code)
	}
	if code := call(t, "POST", b+"/scores", "garbage", map[string]any{"game": "chess", "score": 1}, nil); code != 401 {
		t.Errorf("bad token = %d", code)
	}
	if code := call(t, "POST", b+"/scores", alice, map[string]any{"game": "x y", "score": 1}, nil); code != 400 {
		t.Errorf("bad game = %d", code)
	}
	if code := call(t, "POST", b+"/scores", alice, map[string]any{"game": "chess", "score": -5}, nil); code != 400 {
		t.Errorf("negative score = %d", code)
	}

	// history keeps every submission, newest first, and is per-user
	var hist []map[string]any
	call(t, "GET", b+"/scores/history", alice, nil, &hist)
	if len(hist) != 3 || hist[0]["game"] != "pong" {
		t.Errorf("history = %v", hist)
	}
	call(t, "GET", b+"/scores/history?game=chess", alice, nil, &hist)
	if len(hist) != 2 {
		t.Errorf("chess history len = %d", len(hist))
	}
	call(t, "GET", b+"/scores/history", bob, nil, &hist)
	if len(hist) != 1 {
		t.Errorf("bob history len = %d", len(hist))
	}
}
