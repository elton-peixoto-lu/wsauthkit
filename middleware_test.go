package wsauthkit

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestMiddlewareInjectsClaims(t *testing.T) {
	t.Parallel()

	auth := newTestAuth(t, Config{
		Issuer:     "https://issuer.example.com",
		Audience:   "dashboard",
		SigningKey: []byte("secret"),
	})

	claims := defaultTestClaims()
	claims["role"] = "admin"
	claims["email"] = "user@example.com"
	token := signTestToken(t, []byte("secret"), claims)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ws", nil)
	request.Header.Set("Authorization", "Bearer "+token)

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := MustClaims(r.Context())
		fmt.Fprintf(w, "%s:%s", claims.Subject, claims.MustValue("role"))
	}))

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	if body := recorder.Body.String(); body != "user-123:admin" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestMiddlewareRejectsInvalidAudience(t *testing.T) {
	t.Parallel()

	auth := newTestAuth(t, Config{
		Issuer:     "https://issuer.example.com",
		Audience:   "expected-audience",
		SigningKey: []byte("secret"),
	})

	claims := defaultTestClaims()
	claims["aud"] = "other-audience"
	token := signTestToken(t, []byte("secret"), claims)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ws", nil)
	request.Header.Set("Authorization", "Bearer "+token)

	auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be reached")
	})).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
}

func TestMiddlewareUsesSecWebSocketProtocol(t *testing.T) {
	t.Parallel()

	auth := newTestAuth(t, Config{
		Issuer:     "https://issuer.example.com",
		Audience:   "gateway",
		SigningKey: []byte("secret"),
	})

	claims := jwt.MapClaims{
		"iss": "https://issuer.example.com",
		"aud": "gateway",
		"sub": "socket-user",
		"exp": defaultTestClaims()["exp"],
		"iat": defaultTestClaims()["iat"],
	}
	token := signTestToken(t, []byte("secret"), claims)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ws", nil)
	request.Header.Set("Sec-WebSocket-Protocol", "graphql-ws, bearer, "+token)

	auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(MustClaims(r.Context()).Subject))
	})).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	if body := recorder.Body.String(); body != "socket-user" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestMiddlewareUsesCustomErrorHandler(t *testing.T) {
	t.Parallel()

	auth := newTestAuth(t, Config{
		SigningKey: []byte("secret"),
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, "blocked:"+err.Error(), http.StatusForbidden)
		},
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ws", nil)

	auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be reached")
	})).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
}
