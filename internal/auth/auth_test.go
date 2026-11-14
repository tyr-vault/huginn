package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoginAndMiddleware(t *testing.T) {
	svc, err := New(Config{
		Username: "admin",
		Password: "secret",
		Secret:   []byte("0123456789abcdef"),
		TTL:      time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}

	token, err := svc.Login("admin", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("empty token")
	}
	if _, err := svc.Login("admin", "wrong"); err == nil {
		t.Fatal("expected invalid credentials")
	}

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw := svc.Middleware(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	mw.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
}
