package wsauthkit

import (
	"errors"
	"testing"
)

func TestValidateTokenAcceptsAnyConfiguredIssuer(t *testing.T) {
	t.Parallel()

	auth := newTestAuth(t, Config{
		Issuers:    []string{"https://tenant-a.example.com", "https://tenant-b.example.com"},
		Audience:   "dashboard",
		SigningKey: []byte("secret"),
	})

	claims := defaultTestClaims()
	claims["iss"] = "https://tenant-b.example.com"
	token := signTestToken(t, []byte("secret"), claims)

	validated, err := auth.ValidateToken(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if validated.Issuer != "https://tenant-b.example.com" {
		t.Fatalf("unexpected issuer: %s", validated.Issuer)
	}
}

func TestValidateTokenRejectsUnknownIssuer(t *testing.T) {
	t.Parallel()

	auth := newTestAuth(t, Config{
		Issuers:    []string{"https://tenant-a.example.com"},
		Audience:   "dashboard",
		SigningKey: []byte("secret"),
	})

	claims := defaultTestClaims()
	claims["iss"] = "https://evil.example.com"
	token := signTestToken(t, []byte("secret"), claims)

	_, err := auth.ValidateToken(token)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestValidateTokenAcceptsAnyConfiguredAudience(t *testing.T) {
	t.Parallel()

	auth := newTestAuth(t, Config{
		Issuer:     "https://issuer.example.com",
		Audiences:  []string{"dashboard", "mobile"},
		SigningKey: []byte("secret"),
	})

	claims := defaultTestClaims()
	claims["aud"] = "mobile"
	token := signTestToken(t, []byte("secret"), claims)

	if _, err := auth.ValidateToken(token); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
