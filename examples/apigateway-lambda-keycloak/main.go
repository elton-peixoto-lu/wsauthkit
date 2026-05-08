package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/elton-peixoto-lu/wsauthkit/apigateway"
)

const (
	defaultKeycloakIssuer   = "https://keycloak.example.com/realms/platform"
	defaultKeycloakAudience = "ws-backend"
	defaultKeycloakJWKSURL  = "https://keycloak.example.com/realms/platform/protocol/openid-connect/certs"
)

var (
	keycloakIssuer   = getenvOrDefault("KEYCLOAK_ISSUER", defaultKeycloakIssuer)
	keycloakAudience = getenvOrDefault("KEYCLOAK_AUDIENCE", defaultKeycloakAudience)
	keycloakJWKSURL  = getenvOrDefault("KEYCLOAK_JWKS_URL", defaultKeycloakJWKSURL)
)

func main() {
	auth, err := apigateway.NewAuth(apigateway.Config{
		Issuer:   keycloakIssuer,
		Audience: keycloakAudience,
		JWKSURL:  keycloakJWKSURL,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer auth.Close()

	lambda.Start(newConnectHandler(auth).handleConnect)
}

type connectHandler struct {
	auth *apigateway.Auth
}

func newConnectHandler(auth *apigateway.Auth) *connectHandler {
	return &connectHandler{auth: auth}
}

func (handler *connectHandler) handleConnect(_ context.Context, event events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	claims, err := handler.auth.Authenticate(event)
	if err != nil {
		return apigateway.UnauthorizedResponse(err), nil
	}
	connectionID := event.RequestContext.ConnectionID
	log.Printf("accepted websocket connection id=%s sub=%s", connectionID, claims.Subject)

	// Persist connectionId + claims here if your application needs fan-out, presence, or authorization checks later.
	// Example storage choices: DynamoDB, Redis, Aurora, or another low-latency lookup store.

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "connected",
	}, nil
}

func getenvOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}

	return fallback
}
