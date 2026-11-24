package app

import (
	"context"
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
