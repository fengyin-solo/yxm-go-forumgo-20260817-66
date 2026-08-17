package auth

import (
	"context"
	"testing"
	"time"
)

func TestRegister(t *testing.T) {
	a := NewAuthenticator()
	token, exp, err := a.Register(context.Background(), "user1")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if token == "" {
		t.Error("token should not be empty")
	}
	if exp.Before(time.Now()) {
		t.Error("expiry should be in the future")
	}
}

func TestValidate(t *testing.T) {
	a := NewAuthenticator()
	token, _, err := a.Register(context.Background(), "user1")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	userID, ok := a.Validate(context.Background(), token)
	if !ok {
		t.Fatal("Validate should succeed")
	}
	if userID != "user1" {
		t.Errorf("userID = %q", userID)
	}
}

func TestValidateInvalidToken(t *testing.T) {
	a := NewAuthenticator()
	_, ok := a.Validate(context.Background(), "invalid")
	if ok {
		t.Error("Validate should fail for invalid token")
	}
}

func TestRevoke(t *testing.T) {
	a := NewAuthenticator()
	token, _, err := a.Register(context.Background(), "user1")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := a.Revoke(context.Background(), token); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	_, ok := a.Validate(context.Background(), token)
	if ok {
		t.Error("Validate should fail after revoke")
	}
}

func TestParseAuthorizationHeader(t *testing.T) {
	token, ok := ParseAuthorizationHeader("Bearer abc123")
	if !ok || token != "abc123" {
		t.Errorf("ParseAuthorizationHeader = %q, ok=%v", token, ok)
	}
	_, ok = ParseAuthorizationHeader("bearer abc123")
	if ok {
		t.Error("should require exact Bearer prefix")
	}
	_, ok = ParseAuthorizationHeader("Basic abc123")
	if ok {
		t.Error("should reject Basic auth")
	}
}

func TestValidateToken(t *testing.T) {
	if !ValidateToken("abc", "abc") {
		t.Error("ValidateToken should succeed for matching tokens")
	}
	if ValidateToken("abc", "def") {
		t.Error("ValidateToken should fail for non-matching tokens")
	}
}
