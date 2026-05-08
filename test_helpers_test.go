package wsauthkit

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func newTestAuth(t *testing.T, cfg Config) *Auth {
	t.Helper()

	auth, err := NewAuth(cfg)
	if err != nil {
		t.Fatalf("new auth: %v", err)
	}

	t.Cleanup(func() {
		if err := auth.Close(); err != nil {
			t.Fatalf("close auth: %v", err)
		}
	})

	return auth
}

func signTestToken(t *testing.T, secret []byte, claims jwt.MapClaims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return signedToken
}

func defaultTestClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"iss": "https://issuer.example.com",
		"aud": "dashboard",
		"sub": "user-123",
		"exp": time.Now().Add(5 * time.Minute).Unix(),
		"iat": time.Now().Add(-1 * time.Minute).Unix(),
	}
}
