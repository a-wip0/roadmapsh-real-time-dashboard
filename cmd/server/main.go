package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/a-wip0/roadmapsh-real-time-dashboard/internal/app"
)

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	cfg := app.Config{
		RedisURL:  env("REDIS_URL", "redis://localhost:6379/0"),
		DBPath:    env("DB_PATH", "leaderboard.db"),
		JWTSecret: env("JWT_SECRET", "dev-secret-change-me"),
	}
	srv, err := app.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer srv.Close()

	hs := &http.Server{Addr: env("ADDR", ":8080"), Handler: srv.Handler()}
	go func() {
		log.Printf("listening on %s", hs.Addr)
		if err := hs.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	hs.Shutdown(ctx)
}
