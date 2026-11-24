package app

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var nameRe = regexp.MustCompile(`^[A-Za-z0-9_.-]{3,32}$`)

type creds struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (a *App) token(user string) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   user,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	}).SignedString([]byte(a.cfg.JWTSecret))
}

func (a *App) register(w http.ResponseWriter, r *http.Request) {
	var c creds
	if !decode(w, r, &c) {
		return
	}
	if !nameRe.MatchString(c.Username) || len(c.Password) < 8 || len(c.Password) > 72 {
		errJSON(w, 400, "username must be 3-32 chars [A-Za-z0-9_.-]; password 8-72 chars")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)
	if err != nil {
		errJSON(w, 500, "internal error")
		return
	}
	if _, err := a.db.Exec(`INSERT INTO users(username, password_hash) VALUES(?,?)`, c.Username, string(hash)); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			errJSON(w, 409, "username taken")
			return
		}
		errJSON(w, 500, "internal error")
		return
	}
	writeJSON(w, 201, map[string]string{"username": c.Username})
}

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	var c creds
	if !decode(w, r, &c) {
		return
	}
	var hash string
	err := a.db.QueryRow(`SELECT password_hash FROM users WHERE username=?`, c.Username).Scan(&hash)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(c.Password)) != nil {
		errJSON(w, 401, "invalid credentials")
		return
	}
	t, err := a.token(c.Username)
	if err != nil {
		errJSON(w, 500, "internal error")
		return
	}
	writeJSON(w, 200, map[string]string{"token": t})
}

func (a *App) auth(next func(http.ResponseWriter, *http.Request, string)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok {
			errJSON(w, 401, "missing bearer token")
			return
		}
		var claims jwt.RegisteredClaims
		_, err := jwt.ParseWithClaims(raw, &claims, func(*jwt.Token) (any, error) {
			return []byte(a.cfg.JWTSecret), nil
		}, jwt.WithValidMethods([]string{"HS256"}))
		if err != nil || claims.Subject == "" {
			errJSON(w, 401, "invalid token")
			return
		}
		next(w, r, claims.Subject)
	})
}
