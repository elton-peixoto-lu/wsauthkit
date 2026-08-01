package echowsauth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"

	"github.com/elton-peixoto-lu/wsauthkit"
)

func newTestAuth(t *testing.T) *wsauthkit.Auth {
	t.Helper()

	auth, err := wsauthkit.NewAuth(wsauthkit.Config{
		Issuer:     "https://issuer.example.com",
		Audience:   "dashboard",
		SigningKey: []byte("secret"),
	})
	if err != nil {
		t.Fatalf("new auth: %v", err)
	}
	t.Cleanup(func() { _ = auth.Close() })

	return auth
}

func signToken(t *testing.T) string {
	t.Helper()

	claims := jwt.MapClaims{
		"iss": "https://issuer.example.com",
		"aud": "dashboard",
		"sub": "user-123",
		"exp": time.Now().Add(5 * time.Minute).Unix(),
		"iat": time.Now().Add(-1 * time.Minute).Unix(),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return token
}

func newTestEcho(auth *wsauthkit.Auth) *echo.Echo {
	e := echo.New()
	e.GET("/ws", func(c echo.Context) error {
		claims := wsauthkit.MustClaims(c.Request().Context())
		return c.String(http.StatusOK, claims.Subject)
	}, Middleware(auth))

	return e
}

func TestMiddlewareAuthenticatesValidToken(t *testing.T) {
	t.Parallel()

	e := newTestEcho(newTestAuth(t))

	request := httptest.NewRequest(http.MethodGet, "/ws", nil)
	request.Header.Set("Authorization", "Bearer "+signToken(t))
	recorder := httptest.NewRecorder()

	e.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if recorder.Body.String() != "user-123" {
		t.Fatalf("unexpected body: %q", recorder.Body.String())
	}
}

func TestMiddlewareRejectsMissingToken(t *testing.T) {
	t.Parallel()

	e := newTestEcho(newTestAuth(t))

	request := httptest.NewRequest(http.MethodGet, "/ws", nil)
	recorder := httptest.NewRecorder()

	e.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}
