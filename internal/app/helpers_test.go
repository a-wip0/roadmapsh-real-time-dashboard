package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// newTestApp connects to a throwaway Redis DB (flushed!) and a temp SQLite file.
func newTestApp(t *testing.T) *App {
	t.Helper()
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		url = "redis://localhost:6379/15"
	}
	a, err := New(Config{RedisURL: url, DBPath: filepath.Join(t.TempDir(), "t.db"), JWTSecret: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.rdb.FlushDB(context.Background()).Err(); err != nil {
		a.Close()
		t.Skipf("redis unavailable: %v", err)
	}
	t.Cleanup(a.Close)
	return a
}

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(newTestApp(t).Handler())
	t.Cleanup(ts.Close)
	return ts
}

// call sends an optional JSON body and bearer token, decodes the JSON reply into out.
func call(t *testing.T, method, url, token string, body any, out any) int {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, url, &buf)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if out != nil {
		json.NewDecoder(resp.Body).Decode(out)
	}
	return resp.StatusCode
}

// signup registers and logs in a user, returning the bearer token.
func signup(t *testing.T, base, user string) string {
	t.Helper()
	c := map[string]string{"username": user, "password": "password123"}
	if code := call(t, "POST", base+"/auth/register", "", c, nil); code != 201 {
		t.Fatalf("register %s: %d", user, code)
	}
	var tok struct{ Token string }
	if code := call(t, "POST", base+"/auth/login", "", c, &tok); code != 200 {
		t.Fatalf("login: %d", code)
	}
	return tok.Token
}
