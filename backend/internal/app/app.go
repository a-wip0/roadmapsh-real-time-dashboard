// Package app implements the real-time leaderboard service.
package app

import (
	"database/sql"
	"fmt"

	"github.com/redis/go-redis/v9"
	_ "modernc.org/sqlite"
)

type Config struct {
	RedisURL  string
	DBPath    string
	JWTSecret string
}

type App struct {
	cfg   Config
	rdb   *redis.Client
	db    *sql.DB
	score *redis.Script
	hub   *hub
}

func New(cfg Config) (*App, error) {
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("redis url: %w", err)
	}
	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // SQLite: single writer
	a := &App{cfg: cfg, rdb: redis.NewClient(opt), db: db, score: redis.NewScript(submitLua), hub: newHub()}
	if err := a.migrate(); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *App) Close() { a.rdb.Close(); a.db.Close() }

func (a *App) migrate() error {
	_, err := a.db.Exec(`
CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  username TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS scores (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  username TEXT NOT NULL,
  game TEXT NOT NULL,
  score INTEGER NOT NULL,
  created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_scores_time ON scores(created_at);
CREATE INDEX IF NOT EXISTS idx_scores_user ON scores(username, created_at);`)
	return err
}
