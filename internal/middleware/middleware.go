package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"github.com/example/forumgo/internal/auth"
	"github.com/example/forumgo/internal/logger"
)

// RequestID middleware adds a unique request ID to the context.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			buf := make([]byte, 8)
			rand.Read(buf) // #nosec G404
			id = hex.EncodeToString(buf)
		}
		ctx := context.WithValue(r.Context(), ctxKeyRequestID{}, id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Logging middleware logs requests using slog.
func Logging(next http.Handler, log *logger.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sr := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sr, r)
		log.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sr.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"client_ip", clientIP(r),
		)
	})
}

// Recovery middleware catches panics and returns 500.
func Recovery(next http.Handler, log *logger.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				log.Error("panic", "panic", p, "stack", string(debug.Stack()))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// RequireAuth middleware requires a valid bearer token.
func RequireAuth(next http.Handler, authenticator *auth.Authenticator, log *logger.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := auth.ParseAuthorizationHeader(r.Header.Get("Authorization"))
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		userID, ok := authenticator.Validate(r.Context(), token)
		if !ok {
			writeError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}
		ctx := context.WithValue(r.Context(), ctxKeyUserID{}, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Timeout middleware enforces a per-request deadline.
func Timeout(next http.Handler, d time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), d)
		defer cancel()
		sr := &statusRecorder{ResponseWriter: w, status: http.StatusOK, mu: &sync.Mutex{}}
		done := make(chan struct{})
		go func() {
			next.ServeHTTP(sr, r.WithContext(ctx))
			close(done)
		}()
		select {
		case <-done:
			// normal completion
		case <-ctx.Done():
			sr.WriteHeader(http.StatusServiceUnavailable)
			sr.Write([]byte(`{"error":"request timeout"}`))
		}
	})
}

func newRequestID() string {
	buf := make([]byte, 8)
	rand.Read(buf) // #nosec G404
	return hex.EncodeToString(buf)
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	return r.RemoteAddr
}

type ctxKeyRequestID struct{}
type ctxKeyUserID  struct{}

// RequestIDFrom returns the request ID from the context.
func RequestIDFrom(ctx context.Context) string {
	if v := ctx.Value(ctxKeyRequestID{}); v != nil {
		return v.(string)
	}
	return ""
}

// UserIDFrom returns the authenticated user ID from the context.
func UserIDFrom(ctx context.Context) string {
	if v := ctx.Value(ctxKeyUserID{}); v != nil {
		return v.(string)
	}
	return ""
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	mu     *sync.Mutex
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

func (sr *statusRecorder) Write(b []byte) (int, error) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	return sr.ResponseWriter.Write(b)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}


