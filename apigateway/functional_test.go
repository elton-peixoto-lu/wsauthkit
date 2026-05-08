//go:build functional
// +build functional

package apigateway

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MicahParks/jwkset"
	"github.com/aws/aws-lambda-go/events"
	"github.com/golang-jwt/jwt/v5"
)

func TestFunctionalAuthenticateWithRemoteJWKS(t *testing.T) {
	t.Parallel()

	privateKey, jwksURL := newGatewayJWKSServer(t)
	auth, err := NewAuth(Config{
		Issuer:   "https://keycloak.example.com/realms/platform",
		Audience: "ws-backend",
		JWKSURL:  jwksURL,
	})
	if err != nil {
		t.Fatalf("new auth: %v", err)
	}
	t.Cleanup(func() {
		if err := auth.Close(); err != nil {
			t.Fatalf("close auth: %v", err)
		}
	})

	token := signGatewayRSAToken(t, privateKey, "gateway-key", jwt.MapClaims{
		"iss":                "https://keycloak.example.com/realms/platform",
		"aud":                "ws-backend",
		"sub":                "gateway-user",
		"preferred_username": "alice",
		"exp":                time.Now().Add(5 * time.Minute).Unix(),
		"iat":                time.Now().Add(-1 * time.Minute).Unix(),
	})

	claims, err := auth.Authenticate(events.APIGatewayWebsocketProxyRequest{
		Headers: map[string]string{
			"Sec-WebSocket-Protocol": "graphql-ws, bearer, " + token,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if claims.Subject != "gateway-user" {
		t.Fatalf("unexpected subject: %s", claims.Subject)
	}
	if username, ok := claims.Value("preferred_username"); !ok || username != "alice" {
		t.Fatalf("unexpected preferred_username: %#v", claims.Values)
	}
}

func TestFunctionalAuthenticateWithCustomQueryParameterName(t *testing.T) {
	t.Parallel()

	privateKey, jwksURL := newGatewayJWKSServer(t)
	auth, err := NewAuth(Config{
		Issuer:              "https://keycloak.example.com/realms/platform",
		Audience:            "ws-backend",
		JWKSURL:             jwksURL,
		QueryParameterNames: []string{"authToken"},
	})
	if err != nil {
		t.Fatalf("new auth: %v", err)
	}
	t.Cleanup(func() {
		if err := auth.Close(); err != nil {
			t.Fatalf("close auth: %v", err)
		}
	})

	token := signGatewayRSAToken(t, privateKey, "gateway-key", jwt.MapClaims{
		"iss": "https://keycloak.example.com/realms/platform",
		"aud": "ws-backend",
		"sub": "query-user",
		"exp": time.Now().Add(5 * time.Minute).Unix(),
		"iat": time.Now().Add(-1 * time.Minute).Unix(),
	})

	claims, err := auth.Authenticate(events.APIGatewayWebsocketProxyRequest{
		QueryStringParameters: map[string]string{
			"authToken": token,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if claims.Subject != "query-user" {
		t.Fatalf("unexpected subject: %s", claims.Subject)
	}
}

func newGatewayJWKSServer(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	jwkStorage := jwkset.NewMemoryStorage()
	publicJWK, err := jwkset.NewJWKFromKey(privateKey.Public(), jwkset.JWKOptions{
		Metadata: jwkset.JWKMetadataOptions{
			ALG: jwkset.AlgRS256,
			KID: "gateway-key",
			USE: jwkset.UseSig,
		},
	})
	if err != nil {
		t.Fatalf("create jwk: %v", err)
	}
	if err := jwkStorage.KeyWrite(context.Background(), publicJWK); err != nil {
		t.Fatalf("write jwk: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, err := jwkStorage.JSONPublic(r.Context())
		if err != nil {
			t.Errorf("jwks json: %v", err)
			http.Error(w, "jwks unavailable", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(payload)
	}))

	t.Cleanup(server.Close)

	return privateKey, server.URL
}

func signGatewayRSAToken(t *testing.T, privateKey *rsa.PrivateKey, keyID string, claims jwt.MapClaims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID

	signed, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return signed
}
