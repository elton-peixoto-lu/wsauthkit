//go:build integration
// +build integration

package main

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/golang-jwt/jwt/v5"

	"github.com/elton-peixoto-lu/wsauthkit/apigateway"
)

func TestIntegrationLambdaConnectHandlerAcceptsAuthorizationHeader(t *testing.T) {
	t.Parallel()

	auth := newIntegrationAuth(t)
	handler := newConnectHandler(auth)
	token := signIntegrationToken(t, []byte("integration-secret"), jwt.MapClaims{
		"iss": "https://keycloak.example.com/realms/platform",
		"aud": "ws-backend",
		"sub": "integration-user",
		"exp": time.Now().Add(5 * time.Minute).Unix(),
		"iat": time.Now().Add(-1 * time.Minute).Unix(),
	})

	response, err := handler.handleConnect(context.Background(), events.APIGatewayWebsocketProxyRequest{
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
		RequestContext: events.APIGatewayWebsocketProxyRequestContext{
			ConnectionID: "connection-integration",
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

func TestIntegrationLambdaConnectHandlerRejectsMissingToken(t *testing.T) {
	t.Parallel()

	auth := newIntegrationAuth(t)
	handler := newConnectHandler(auth)

	response, err := handler.handleConnect(context.Background(), events.APIGatewayWebsocketProxyRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unexpected status: %d", response.StatusCode)
	}
	if response.Body != "authentication token missing" {
		t.Fatalf("unexpected body: %q", response.Body)
	}
}

func newIntegrationAuth(t *testing.T) *apigateway.Auth {
	t.Helper()

	auth, err := apigateway.NewAuth(apigateway.Config{
		Issuer:     keycloakIssuer,
		Audience:   keycloakAudience,
		SigningKey: []byte("integration-secret"),
	})
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

func signIntegrationToken(t *testing.T, signingKey []byte, claims jwt.MapClaims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(signingKey)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return signedToken
}
