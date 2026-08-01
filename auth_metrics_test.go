package wsauthkit

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthenticateCallsOnAuthResult(t *testing.T) {
	t.Parallel()

	var gotErr error
	var called bool

	auth := newTestAuth(t, Config{
		Issuer:     "https://issuer.example.com",
		Audience:   "dashboard",
		SigningKey: []byte("secret"),
		OnAuthResult: func(_ *http.Request, claims *Claims, err error) {
			called = true
			gotErr = err
			if err == nil && claims == nil {
				t.Fatal("expected claims on success")
			}
		},
	})

	token := signTestToken(t, []byte("secret"), defaultTestClaims())
	request := httptest.NewRequest("GET", "/ws", nil)
	request.Header.Set("Authorization", "Bearer "+token)

	if _, err := auth.Authenticate(request); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Fatal("expected OnAuthResult to be called")
	}
	if gotErr != nil {
		t.Fatalf("unexpected error passed to OnAuthResult: %v", gotErr)
	}
}
