package wsauthkit

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestValidateTokenAcceptsValidToken(t *testing.T) {
	t.Parallel()

	auth := newTestAuth(t, Config{
		Issuer:     "https://issuer.example.com",
		Audience:   "dashboard",
		SigningKey: []byte("secret"),
	})

	claims := defaultTestClaims()
	claims["role"] = "admin"
	token := signTestToken(t, []byte("secret"), claims)

	validatedClaims, err := auth.ValidateToken(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if validatedClaims.Subject != "user-123" {
		t.Fatalf("unexpected subject: %s", validatedClaims.Subject)
	}
	if role, ok := validatedClaims.Value("role"); !ok || role != "admin" {
		t.Fatalf("unexpected custom claim: %#v", validatedClaims.Values)
	}
}

func TestValidateTokenRejectsExpiredToken(t *testing.T) {
	t.Parallel()

	auth := newTestAuth(t, Config{
		Issuer:     "https://issuer.example.com",
		Audience:   "dashboard",
		SigningKey: []byte("secret"),
	})

	expiredClaims := defaultTestClaims()
	expiredClaims["exp"] = time.Now().Add(-2 * time.Minute).Unix()
	token := signTestToken(t, []byte("secret"), expiredClaims)

	_, err := auth.ValidateToken(token)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestNewJWTValidatorSupportsCustomKeyFunc(t *testing.T) {
	t.Parallel()

	validator, closer, err := NewJWTValidator(Config{
		Issuer:   "https://issuer.example.com",
		Audience: "dashboard",
		KeyFunc: func(_ *jwt.Token) (any, error) {
			return []byte("secret"), nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if closer != nil {
		t.Fatal("expected nil closer for custom key func")
	}

	claims := defaultTestClaims()
	token := signTestToken(t, []byte("secret"), claims)

	validatedClaims, err := validator.ValidateToken(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if validatedClaims.Issuer != "https://issuer.example.com" {
		t.Fatalf("unexpected issuer: %s", validatedClaims.Issuer)
	}
}
