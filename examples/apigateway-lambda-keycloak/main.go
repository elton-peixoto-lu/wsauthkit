package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/elton-peixoto-lu/wsauthkit/apigateway"
)

const (
	keycloakIssuer   = "https://keycloak.example.com/realms/platform"
	keycloakAudience = "ws-backend"
	keycloakJWKSURL  = "https://keycloak.example.com/realms/platform/protocol/openid-connect/certs"
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
