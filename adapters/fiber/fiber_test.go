package fiberwsauth

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

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

func newTestApp(auth *wsauthkit.Auth) *fiber.App {
	app := fiber.New()
	app.Get("/ws", Middleware(auth), func(c *fiber.Ctx) error {
		claims, ok := ClaimsFromFiber(c)
		if !ok {
			return fiber.NewError(http.StatusInternalServerError, "missing claims")
		}
		return c.SendString(claims.Subject)
	})

	return app
}

func TestMiddlewareAuthenticatesValidToken(t *testing.T) {
	t.Parallel()

	app := newTestApp(newTestAuth(t))

	request := httptest.NewRequest(http.MethodGet, "/ws", nil)
	request.Header.Set("Authorization", "Bearer "+signToken(t))

	resp, err := app.Test(request)
	if err != nil {
		t.Fatalf("test request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != "user-123" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestMiddlewareRejectsMissingToken(t *testing.T) {
	t.Parallel()

	app := newTestApp(newTestAuth(t))

	request := httptest.NewRequest(http.MethodGet, "/ws", nil)

	resp, err := app.Test(request)
	if err != nil {
		t.Fatalf("test request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}
