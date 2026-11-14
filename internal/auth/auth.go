package auth

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Config holds single-admin credentials.
type Config struct {
	Username string
	Password string
	Secret   []byte
	TTL      time.Duration
}

// Service issues and validates JWTs for the env admin user.
type Service struct {
	cfg Config
}

func New(cfg Config) (*Service, error) {
	if cfg.Username == "" || cfg.Password == "" {
		return nil, errors.New("ADMIN_USER and ADMIN_PASSWORD are required")
	}
	if len(cfg.Secret) < 16 {
		return nil, errors.New("JWT_SECRET must be at least 16 characters")
	}
	if cfg.TTL <= 0 {
		cfg.TTL = 24 * time.Hour
	}
	return &Service{cfg: cfg}, nil
}

type claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Login validates credentials and returns a signed JWT.
func (s *Service) Login(username, password string) (string, error) {
	userOK := subtle.ConstantTimeCompare([]byte(username), []byte(s.cfg.Username)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(password), []byte(s.cfg.Password)) == 1
	if !userOK || !passOK {
		return "", errors.New("invalid credentials")
	}
	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.TTL)),
		},
	})
	return token.SignedString(s.cfg.Secret)
}

// Middleware requires a valid Bearer JWT.
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		raw := strings.TrimPrefix(header, "Bearer ")
		parsed, err := jwt.ParseWithClaims(raw, &claims{}, func(t *jwt.Token) (any, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return s.cfg.Secret, nil
		})
		if err != nil || !parsed.Valid {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
