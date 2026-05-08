package wsauthkit

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewAuthRequiresExactlyOneKeySource(t *testing.T) {
	t.Parallel()

	_, err := NewAuth(Config{})
	if !errors.Is(err, ErrMissingKeySource) {
		t.Fatalf("expected ErrMissingKeySource, got %v", err)
	}

	_, err = NewAuth(Config{
		SigningKey: []byte("secret"),
		JWKSURL:    "https://issuer.example.com/jwks.json",
	})
	if err == nil {
		t.Fatal("expected conflict error for multiple key sources")
	}
}

func TestAuthenticateUsesConfiguredExtractorAndValidator(t *testing.T) {
	t.Parallel()

	auth := &Auth{
		extractor: TokenExtractorFunc(func(_ *http.Request) (string, error) {
			return "token-value", nil
		}),
		validator: tokenValidatorFunc(func(token string) (*Claims, error) {
			if token != "token-value" {
				t.Fatalf("unexpected token: %s", token)
			}
			return &Claims{Values: map[string]any{"role": "admin"}}, nil
		}),
		errorHandler: DefaultErrorHandler,
	}

	claims, err := auth.Authenticate(httptest.NewRequest(http.MethodGet, "/ws", nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if role, ok := claims.Value("role"); !ok || role != "admin" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestNilAuthMiddlewareReturnsUnauthorized(t *testing.T) {
	t.Parallel()

	var auth *Auth
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ws", nil)

	auth.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler should not be reached")
	})).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", recorder.Code)
	}
}

type tokenValidatorFunc func(token string) (*Claims, error)

func (validatorFunc tokenValidatorFunc) ValidateToken(token string) (*Claims, error) {
	return validatorFunc(token)
}
