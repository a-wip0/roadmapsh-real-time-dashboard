package app

import "testing"

func TestAuth(t *testing.T) {
	b := newTestServer(t).URL
	creds := map[string]string{"username": "alice", "password": "password123"}

	if code := call(t, "POST", b+"/auth/register", "", creds, nil); code != 201 {
		t.Fatalf("register = %d", code)
	}
	if code := call(t, "POST", b+"/auth/register", "", creds, nil); code != 409 {
		t.Errorf("duplicate register = %d", code)
	}
	for name, c := range map[string]map[string]string{
		"short password": {"username": "bob", "password": "short"},
		"bad username":   {"username": "a b", "password": "password123"},
	} {
		if code := call(t, "POST", b+"/auth/register", "", c, nil); code != 400 {
			t.Errorf("%s = %d", name, code)
		}
	}

	var tok struct{ Token string }
	if code := call(t, "POST", b+"/auth/login", "", creds, &tok); code != 200 || tok.Token == "" {
		t.Errorf("login = %d token=%q", code, tok.Token)
	}
	bad := map[string]string{"username": "alice", "password": "wrongpass1"}
	if code := call(t, "POST", b+"/auth/login", "", bad, nil); code != 401 {
		t.Errorf("bad password = %d", code)
	}
	unknown := map[string]string{"username": "ghost", "password": "password123"}
	if code := call(t, "POST", b+"/auth/login", "", unknown, nil); code != 401 {
		t.Errorf("unknown user = %d", code)
	}
}
