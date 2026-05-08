package wsauthkit

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestClaimsAsMapIncludesRegisteredAndCustomClaims(t *testing.T) {
	t.Parallel()

	expiresAt := time.Now().Add(10 * time.Minute)
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://issuer.example.com",
			Subject:   "user-123",
			Audience:  jwt.ClaimStrings{"dashboard"},
			ID:        "token-id",
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		Values: map[string]any{
			"role": "admin",
		},
	}

	asMap := claims.AsMap()

	if asMap["iss"] != "https://issuer.example.com" {
		t.Fatalf("unexpected issuer: %#v", asMap["iss"])
	}
	if asMap["sub"] != "user-123" {
		t.Fatalf("unexpected subject: %#v", asMap["sub"])
	}
	if asMap["jti"] != "token-id" {
		t.Fatalf("unexpected jti: %#v", asMap["jti"])
	}
	if asMap["role"] != "admin" {
		t.Fatalf("unexpected role: %#v", asMap["role"])
	}

	audience, ok := asMap["aud"].([]string)
	if !ok || len(audience) != 1 || audience[0] != "dashboard" {
		t.Fatalf("unexpected audience: %#v", asMap["aud"])
	}
}

func TestMustValuePanicsWhenClaimIsMissing(t *testing.T) {
	t.Parallel()

	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("expected panic for missing claim")
		}
	}()

	(&Claims{}).MustValue("role")
}

func TestClaimsContextRoundTrip(t *testing.T) {
	t.Parallel()

	expectedClaims := &Claims{Values: map[string]any{"role": "admin"}}
	ctx := WithClaims(context.Background(), expectedClaims)

	actualClaims, ok := ClaimsFromContext(ctx)
	if !ok {
		t.Fatal("expected claims in context")
	}
	if actualClaims != expectedClaims {
		t.Fatalf("expected same claims pointer, got %#v", actualClaims)
	}
}

func TestMustClaimsPanicsWhenMissing(t *testing.T) {
	t.Parallel()

	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("expected panic for missing context claims")
		}
	}()

	MustClaims(context.Background())
}
