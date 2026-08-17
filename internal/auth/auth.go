package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Authenticator manages bearer tokens and their associated user IDs.
type Authenticator struct {
	mu      sync.RWMutex
	tokens  map[string]string // token -> userID
	expires map[string]time.Time
	now     func() time.Time
}

// NewAuthenticator creates a new Authenticator.
func NewAuthenticator() *Authenticator {
	return &Authenticator{
		tokens:  make(map[string]string),
		expires: make(map[string]time.Time),
		now:     time.Now,
	}
}

// Register issues a new bearer token for the given user ID.
func (a *Authenticator) Register(ctx context.Context, userID string) (token string, expiresAt time.Time, err error) {
	if userID == "" {
		return "", time.Time{}, fmt.Errorf("userID required")
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", time.Time{}, err
	}
	token = hex.EncodeToString(buf)
	expiresAt = a.now().Add(24 * time.Hour)
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tokens[token] = userID
	a.expires[token] = expiresAt
	go a.cleanExpired()
	return token, expiresAt, nil
}

// Validate checks a bearer token and returns the associated user ID.
func (a *Authenticator) Validate(ctx context.Context, token string) (userID string, ok bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if userID, ok = a.tokens[token]; !ok {
		return "", false
	}
	if exp, ok := a.expires[token]; ok && a.now().After(exp) {
		return "", false
	}
	return userID, true
}

// Revoke invalidates a bearer token.
func (a *Authenticator) Revoke(ctx context.Context, token string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.tokens[token]; !ok {
		return fmt.Errorf("token not found")
	}
	delete(a.tokens, token)
	delete(a.expires, token)
	return nil
}

func (a *Authenticator) cleanExpired() {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.now()
	for token, exp := range a.expires {
		if now.After(exp) {
			delete(a.tokens, token)
			delete(a.expires, token)
		}
	}
}

// ParseAuthorizationHeader extracts the bearer token from an Authorization header.
func ParseAuthorizationHeader(hdr string) (token string, ok bool) {
	if len(hdr) > 7 && hdr[:7] == "Bearer " {
		return hdr[7:], true
	}
	return "", false
}

// ValidateToken uses constant-time comparison to validate a token.
func ValidateToken(token, valid string) bool {
	return subtle.ConstantTimeCompare([]byte(token), []byte(valid)) == 1
}
