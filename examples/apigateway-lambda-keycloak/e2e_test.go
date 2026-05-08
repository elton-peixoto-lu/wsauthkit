//go:build e2e
// +build e2e

package main

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

	"github.com/elton-peixoto-lu/wsauthkit/apigateway"
)

func TestE2ELambdaConnectHandlerWithKeycloakStyleJWKS(t *testing.T) {
	t.Parallel()

	privateKey, jwksURL := newLambdaJWKSServer(t)
	auth, err := apigateway.NewAuth(apigateway.Config{
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

	handler := newConnectHandler(auth)
	token := signLambdaRSAToken(t, privateKey, "lambda-key", jwt.MapClaims{
		"iss":                "https://keycloak.example.com/realms/platform",
		"aud":                "ws-backend",
		"sub":                "lambda-user",
		"preferred_username": "bob",
		"exp":                time.Now().Add(5 * time.Minute).Unix(),
		"iat":                time.Now().Add(-1 * time.Minute).Unix(),
	})

	response, err := handler.handleConnect(context.Background(), events.APIGatewayWebsocketProxyRequest{
		Headers: map[string]string{
			"Sec-WebSocket-Protocol": "graphql-ws, bearer, " + token,
		},
		RequestContext: events.APIGatewayWebsocketProxyRequestContext{
			ConnectionID: "connection-123",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", response.StatusCode)
	}
	if response.Body != "connected" {
		t.Fatalf("unexpected body: %q", response.Body)
	}
}

func TestE2ELambdaConnectHandlerRejectsInvalidToken(t *testing.T) {
	t.Parallel()

	privateKey, jwksURL := newLambdaJWKSServer(t)
	auth, err := apigateway.NewAuth(apigateway.Config{
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

	handler := newConnectHandler(auth)
	token := signLambdaRSAToken(t, privateKey, "wrong-key", jwt.MapClaims{
		"iss": "https://keycloak.example.com/realms/platform",
		"aud": "ws-backend",
		"sub": "bad-user",
		"exp": time.Now().Add(5 * time.Minute).Unix(),
		"iat": time.Now().Add(-1 * time.Minute).Unix(),
	})

	response, err := handler.handleConnect(context.Background(), events.APIGatewayWebsocketProxyRequest{
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unexpected status: %d", response.StatusCode)
	}
	if response.Body != "invalid authentication token" {
		t.Fatalf("unexpected body: %q", response.Body)
	}
}

func newLambdaJWKSServer(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	jwkStorage := jwkset.NewMemoryStorage()
	publicJWK, err := jwkset.NewJWKFromKey(privateKey.Public(), jwkset.JWKOptions{
		Metadata: jwkset.JWKMetadataOptions{
			ALG: jwkset.AlgRS256,
			KID: "lambda-key",
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

func signLambdaRSAToken(t *testing.T, privateKey *rsa.PrivateKey, keyID string, claims jwt.MapClaims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID

	signed, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return signed
}
