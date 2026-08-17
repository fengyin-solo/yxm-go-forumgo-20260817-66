package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/forumgo/internal/auth"
	"github.com/example/forumgo/internal/logger"
)

func TestRequestID(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		id := RequestIDFrom(r.Context())
		w.Write([]byte(id))
	})
	handler := RequestID(mux)

	// Without X-Request-ID header
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("X-Request-ID should be set")
	}

	// With X-Request-ID header
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("X-Request-ID", "my-id")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Header().Get("X-Request-ID") != "my-id" {
		t.Errorf("X-Request-ID = %q", rec2.Header().Get("X-Request-ID"))
	}
}

func TestRecovery(t *testing.T) {
	var buf strings.Builder
	log := logger.New(&buf, logger.LevelError)
	mux := http.NewServeMux()
	mux.HandleFunc("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})
	handler := Recovery(mux, log)
	req := httptest.NewRequest("GET", "/panic", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestRequireAuth(t *testing.T) {
	var buf strings.Builder
	log := logger.New(&buf, logger.LevelError)
	a := auth.NewAuthenticator()
	token, _, _ := a.Register(nil, "user1")

	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	handler := RequireAuth(mux, a, log)

	// Without auth
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("without auth: status = %d", rec.Code)
	}

	// With valid token
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("with auth: status = %d", rec2.Code)
	}
}

func TestTimeout(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.Write([]byte("ok"))
	})
	handler := Timeout(mux, 10*time.Millisecond)
	req := httptest.NewRequest("GET", "/slow", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("timeout: status = %d (want 503)", rec.Code)
	}
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(3, 1*time.Minute)
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	handler := rl.Limit(mux)

	// First 3 requests should succeed
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("request %d: status = %d", i+1, rec.Code)
		}
	}

	// 4th request should be rate-limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("4th request: status = %d (want 429)", rec.Code)
	}
}

func TestUserIDFrom(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxKeyUserID{}, "test-user")
	if UserIDFrom(ctx) != "test-user" {
		t.Error("UserIDFrom should return the user ID")
	}
	if UserIDFrom(context.Background()) != "" {
		t.Error("UserIDFrom with empty context should return empty string")
	}
}
