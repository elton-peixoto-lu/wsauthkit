package wsauthkit

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuthenticateRejectsRevokedToken(t *testing.T) {
	t.Parallel()

	secret := []byte("secret")
	auth := newTestAuth(t, Config{
		SigningKey: secret,
		Revoker: RevokerFunc(func(_ context.Context, claims *Claims) (bool, error) {
			return claims.Subject == "user-123", nil
		}),
	})

	token := signTestToken(t, secret, defaultTestClaims())

	request := httptest.NewRequest("GET", "/ws", nil)
	request.Header.Set("Authorization", "Bearer "+token)

	_, err := auth.Authenticate(request)
	if err != ErrTokenRevoked {
		t.Fatalf("expected ErrTokenRevoked, got %v", err)
	}
}

func TestReverifyDetectsExpiryAndRevocation(t *testing.T) {
	t.Parallel()

	revoked := make(chan struct{})
	auth := newTestAuth(t, Config{
		SigningKey: []byte("secret"),
		Revoker: RevokerFunc(func(_ context.Context, _ *Claims) (bool, error) {
			select {
			case <-revoked:
				return true, nil
			default:
				return false, nil
			}
		}),
	})

	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go auth.Reverify(ctx, claims, 10*time.Millisecond, func(err error) {
		done <- err
	})

	close(revoked)

	select {
	case err := <-done:
		if err != ErrTokenRevoked {
			t.Fatalf("expected ErrTokenRevoked, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for reverify to detect revocation")
	}
}
