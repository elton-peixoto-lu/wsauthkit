package wsauthkit

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestComparisonManualVsWSAuthKitAuthenticatedFlow(t *testing.T) {
	t.Parallel()

	signingKey := []byte("secret")
	tokenClaims := defaultTestClaims()
	tokenClaims["sub"] = "compare-user"
	tokenClaims["role"] = "admin"
	token := signTestToken(t, signingKey, tokenClaims)

	manualRecorder := httptest.NewRecorder()
	manualRequest := httptest.NewRequest(http.MethodGet, "/ws", nil)
	manualRequest.Header.Set("Authorization", "Bearer "+token)

	manualHandler := manualHandshakeAuthMiddleware(signingKey, "https://issuer.example.com", "dashboard", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(manualClaimsContextKey{}).(*Claims)
		fmt.Fprintf(w, "manual user=%s role=%s", claims.Subject, claims.MustValue("role"))
	}))

	manualHandler.ServeHTTP(manualRecorder, manualRequest)

	auth := newTestAuth(t, Config{
		Issuer:     "https://issuer.example.com",
		Audience:   "dashboard",
		SigningKey: signingKey,
	})

	kitRecorder := httptest.NewRecorder()
	kitRequest := httptest.NewRequest(http.MethodGet, "/ws", nil)
	kitRequest.Header.Set("Authorization", "Bearer "+token)

	kitHandler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := MustClaims(r.Context())
		fmt.Fprintf(w, "kit user=%s role=%s", claims.Subject, claims.MustValue("role"))
	}))

	kitHandler.ServeHTTP(kitRecorder, kitRequest)

	if manualRecorder.Code != http.StatusOK {
		t.Fatalf("manual flow status = %d", manualRecorder.Code)
	}
	if kitRecorder.Code != http.StatusOK {
		t.Fatalf("kit flow status = %d", kitRecorder.Code)
	}

	t.Logf("manual flow response: %s", manualRecorder.Body.String())
	t.Logf("wsauthkit flow response: %s", kitRecorder.Body.String())
}

func TestComparisonManualVsWSAuthKitInvalidToken(t *testing.T) {
	t.Parallel()

	signingKey := []byte("secret")
	invalidToken := signTestToken(t, []byte("wrong-secret"), jwt.MapClaims{
		"iss": "https://issuer.example.com",
		"aud": "dashboard",
		"sub": "bad-user",
		"exp": defaultTestClaims()["exp"],
		"iat": defaultTestClaims()["iat"],
	})

	manualRecorder := httptest.NewRecorder()
	manualRequest := httptest.NewRequest(http.MethodGet, "/ws", nil)
	manualRequest.Header.Set("Authorization", "Bearer "+invalidToken)

	manualHandler := manualHandshakeAuthMiddleware(signingKey, "https://issuer.example.com", "dashboard", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("manual handler should not be reached")
	}))

	manualHandler.ServeHTTP(manualRecorder, manualRequest)

	auth := newTestAuth(t, Config{
		Issuer:     "https://issuer.example.com",
		Audience:   "dashboard",
		SigningKey: signingKey,
	})

	kitRecorder := httptest.NewRecorder()
	kitRequest := httptest.NewRequest(http.MethodGet, "/ws", nil)
	kitRequest.Header.Set("Authorization", "Bearer "+invalidToken)

	kitHandler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("kit handler should not be reached")
	}))

	kitHandler.ServeHTTP(kitRecorder, kitRequest)

	if manualRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("manual flow status = %d", manualRecorder.Code)
	}
	if kitRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("kit flow status = %d", kitRecorder.Code)
	}

	t.Logf("manual invalid response: %q", manualRecorder.Body.String())
	t.Logf("wsauthkit invalid response: %q", kitRecorder.Body.String())
}

type manualClaimsContextKey struct{}

func manualHandshakeAuthMiddleware(signingKey []byte, expectedIssuer string, expectedAudience string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawHeader := r.Header.Get("Authorization")
		if rawHeader == "" {
			http.Error(w, "authentication token missing", http.StatusUnauthorized)
			return
		}

		tokenValue, err := AuthorizationHeaderExtractor().ExtractToken(r)
		if err != nil {
			http.Error(w, "malformed authentication token", http.StatusUnauthorized)
			return
		}

		parser := jwt.NewParser(
			jwt.WithIssuer(expectedIssuer),
			jwt.WithAudience(expectedAudience),
			jwt.WithExpirationRequired(),
			jwt.WithIssuedAt(),
		)

		claimsMap := jwt.MapClaims{}
		parsedToken, err := parser.ParseWithClaims(tokenValue, claimsMap, func(_ *jwt.Token) (any, error) {
			return signingKey, nil
		})
		if err != nil || !parsedToken.Valid {
			http.Error(w, "invalid authentication token", http.StatusUnauthorized)
			return
		}

		claims := claimsFromMapClaims(claimsMap)
		ctx := context.WithValue(r.Context(), manualClaimsContextKey{}, claims)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
