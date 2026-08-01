//go:build functional
// +build functional

package wsauthkit

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MicahParks/jwkset"
	"github.com/golang-jwt/jwt/v5"
)

func TestFunctionalAuthenticateWithRemoteJWKS(t *testing.T) {
	t.Parallel()

	privateKey, jwksServerURL := newRemoteJWKSTestServer(t)

	auth := newTestAuth(t, Config{
		Issuer:   "https://issuer.example.com",
		Audience: "dashboard",
		JWKSURL:  jwksServerURL,
	})

	token := signRSATestToken(t, privateKey, "test-key", jwt.MapClaims{
		"iss": "https://issuer.example.com",
		"aud": "dashboard",
		"sub": "remote-user",
		"exp": time.Now().Add(5 * time.Minute).Unix(),
		"iat": time.Now().Add(-1 * time.Minute).Unix(),
	})

	request := httptest.NewRequest(http.MethodGet, "/ws", nil)
	request.Header.Set("Authorization", "Bearer "+token)

	claims, err := auth.Authenticate(request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.Subject != "remote-user" {
		t.Fatalf("unexpected subject: %s", claims.Subject)
	}
}

func TestFunctionalAuthenticateWithDexStyleJWKS(t *testing.T) {
	t.Parallel()

	privateKey, jwksServerURL := newDexStyleJWKSTestServer(t)

	auth := newTestAuth(t, Config{
		Issuer:   "https://dex.example.com",
		Audience: "ws-backend",
		JWKSURL:  jwksServerURL,
	})

	token := signRSATestToken(t, privateKey, "test-key", jwt.MapClaims{
		"iss": "https://dex.example.com",
		"aud": "ws-backend",
		"sub": "dex-user",
		"exp": time.Now().Add(5 * time.Minute).Unix(),
		"iat": time.Now().Add(-1 * time.Minute).Unix(),
	})

	request := httptest.NewRequest(http.MethodGet, "/ws", nil)
	request.Header.Set("Authorization", "Bearer "+token)

	claims, err := auth.Authenticate(request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.Subject != "dex-user" {
		t.Fatalf("unexpected subject: %s", claims.Subject)
	}
}

func TestFunctionalDefaultErrorResponses(t *testing.T) {
	t.Parallel()

	auth := newTestAuth(t, Config{
		Issuer:     "https://issuer.example.com",
		Audience:   "dashboard",
		SigningKey: []byte("secret"),
	})

	handler := auth.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler should not be reached")
	}))

	testCases := []struct {
		name           string
		headerName     string
		headerValue    string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "missing token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "authentication token missing\n",
		},
		{
			name:           "malformed authorization header",
			headerName:     "Authorization",
			headerValue:    "Bearer too many parts here",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "malformed authentication token\n",
		},
		{
			name:           "invalid token signature",
			headerName:     "Authorization",
			headerValue:    "Bearer " + invalidSignedToken(t),
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "invalid authentication token\n",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/ws", nil)
			if testCase.headerName != "" {
				request.Header.Set(testCase.headerName, testCase.headerValue)
			}

			handler.ServeHTTP(recorder, request)

			if recorder.Code != testCase.expectedStatus {
				t.Fatalf("expected status %d, got %d", testCase.expectedStatus, recorder.Code)
			}
			if recorder.Body.String() != testCase.expectedBody {
				t.Fatalf("expected body %q, got %q", testCase.expectedBody, recorder.Body.String())
			}
		})
	}
}

func TestFunctionalDefaultExtractorPrefersAuthorizationHeader(t *testing.T) {
	t.Parallel()

	auth := newTestAuth(t, Config{
		Issuer:     "https://issuer.example.com",
		Audience:   "dashboard",
		SigningKey: []byte("secret"),
	})

	headerClaims := defaultTestClaims()
	headerClaims["sub"] = "header-user"
	headerToken := signTestToken(t, []byte("secret"), headerClaims)

	subprotocolClaims := defaultTestClaims()
	subprotocolClaims["sub"] = "subprotocol-user"
	subprotocolToken := signTestToken(t, []byte("secret"), subprotocolClaims)

	request := httptest.NewRequest(http.MethodGet, "/ws", nil)
	request.Header.Set("Authorization", "Bearer "+headerToken)
	request.Header.Set("Sec-WebSocket-Protocol", "graphql-ws, bearer, "+subprotocolToken)

	claims, err := auth.Authenticate(request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.Subject != "header-user" {
		t.Fatalf("expected authorization header token to win, got %s", claims.Subject)
	}
}

func TestFunctionalStandaloneAPIFlow(t *testing.T) {
	t.Parallel()

	auth := newTestAuth(t, Config{
		Issuer:     "https://issuer.example.com",
		Audience:   "dashboard",
		SigningKey: []byte("secret"),
	})

	requestClaims := defaultTestClaims()
	requestClaims["role"] = "operator"
	token := signTestToken(t, []byte("secret"), requestClaims)

	request := httptest.NewRequest(http.MethodGet, "/ws", nil)
	request.Header.Set("Authorization", "Bearer "+token)

	extractedToken, err := auth.ExtractToken(request)
	if err != nil {
		t.Fatalf("extract token: %v", err)
	}

	validatedClaims, err := auth.ValidateToken(extractedToken)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}

	requestWithClaims := request.WithContext(WithClaims(request.Context(), validatedClaims))
	claimsFromContext, ok := ClaimsFromContext(requestWithClaims.Context())
	if !ok {
		t.Fatal("expected claims in request context")
	}

	if role, ok := claimsFromContext.Value("role"); !ok || role != "operator" {
		t.Fatalf("unexpected context claims: %#v", claimsFromContext.Values)
	}
}

func TestFunctionalCookieExtractionWithOriginValidation(t *testing.T) {
	t.Parallel()

	auth := newTestAuth(t, Config{
		Issuer:          "https://issuer.example.com",
		Audience:        "dashboard",
		SigningKey:      []byte("secret"),
		Extractors:      []TokenExtractor{CookieExtractor("session")},
		OriginValidator: AllowedOrigins("https://app.example.com"),
	})

	token := signTestToken(t, []byte("secret"), defaultTestClaims())

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(MustClaims(r.Context()).Subject))
	}))

	t.Run("trusted origin with cookie succeeds", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/ws", nil)
		request.Header.Set("Origin", "https://app.example.com")
		request.AddCookie(&http.Cookie{Name: "session", Value: token})

		handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
		}
		if recorder.Body.String() != "user-123" {
			t.Fatalf("unexpected body: %q", recorder.Body.String())
		}
	})

	t.Run("untrusted origin is rejected before token is read", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/ws", nil)
		request.Header.Set("Origin", "https://evil.example.com")
		request.AddCookie(&http.Cookie{Name: "session", Value: token})

		handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", recorder.Code)
		}
	})
}

func TestFunctionalRevokerRejectsHandshake(t *testing.T) {
	t.Parallel()

	auth := newTestAuth(t, Config{
		Issuer:     "https://issuer.example.com",
		Audience:   "dashboard",
		SigningKey: []byte("secret"),
		Revoker: RevokerFunc(func(_ context.Context, claims *Claims) (bool, error) {
			return claims.Subject == "user-123", nil
		}),
	})

	token := signTestToken(t, []byte("secret"), defaultTestClaims())

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be reached for a revoked token")
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ws", nil)
	request.Header.Set("Authorization", "Bearer "+token)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func newRemoteJWKSTestServer(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	jwkStorage := jwkset.NewMemoryStorage()
	publicJWK, err := jwkset.NewJWKFromKey(privateKey.Public(), jwkset.JWKOptions{
		Metadata: jwkset.JWKMetadataOptions{
			ALG: jwkset.AlgRS256,
			KID: "test-key",
			USE: jwkset.UseSig,
		},
	})
	if err != nil {
		t.Fatalf("create public jwk: %v", err)
	}

	if err := jwkStorage.KeyWrite(context.Background(), publicJWK); err != nil {
		t.Fatalf("write public jwk: %v", err)
	}

	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonBody, err := jwkStorage.JSONPublic(r.Context())
		if err != nil {
			t.Errorf("marshal jwks: %v", err)
			http.Error(w, "jwks unavailable", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jsonBody)
	}))

	t.Cleanup(jwksServer.Close)

	return privateKey, jwksServer.URL
}

func newDexStyleJWKSTestServer(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	jwkStorage := jwkset.NewMemoryStorage()
	publicJWK, err := jwkset.NewJWKFromKey(privateKey.Public(), jwkset.JWKOptions{
		Metadata: jwkset.JWKMetadataOptions{
			ALG: jwkset.AlgRS256,
			KID: "test-key",
			USE: jwkset.UseSig,
		},
	})
	if err != nil {
		t.Fatalf("create public jwk: %v", err)
	}

	if err := jwkStorage.KeyWrite(context.Background(), publicJWK); err != nil {
		t.Fatalf("write public jwk: %v", err)
	}

	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/keys" {
			http.NotFound(w, r)
			return
		}

		jsonBody, err := jwkStorage.JSONPublic(r.Context())
		if err != nil {
			t.Errorf("marshal jwks: %v", err)
			http.Error(w, "jwks unavailable", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jsonBody)
	}))

	t.Cleanup(jwksServer.Close)

	return privateKey, jwksServer.URL + "/keys"
}

func signRSATestToken(t *testing.T, privateKey *rsa.PrivateKey, keyID string, claims jwt.MapClaims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID

	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("sign rsa token: %v", err)
	}

	return signedToken
}

func invalidSignedToken(t *testing.T) string {
	t.Helper()

	claims := defaultTestClaims()
	validToken := signTestToken(t, []byte("different-secret"), claims)
	return strings.TrimSpace(fmt.Sprintf("%s", validToken))
}
