package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/golang-jwt/jwt/v5"
)

const (
	keycloakIssuer   = "https://keycloak.example.com/realms/platform"
	keycloakAudience = "ws-backend"
	keycloakJWKSURL  = "https://keycloak.example.com/realms/platform/protocol/openid-connect/certs"
)

type websocketClaims struct {
	jwt.RegisteredClaims
	PreferredUsername string `json:"preferred_username"`
}

type connectHandler struct {
	parser  *jwt.Parser
	keyFunc jwt.Keyfunc
	stopJWKS func() error
}

func main() {
	handler, err := newConnectHandler()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := handler.stopJWKS(); err != nil {
			log.Printf("jwks shutdown error: %v", err)
		}
	}()

	lambda.Start(handler.handleConnect)
}

func newConnectHandler() (*connectHandler, error) {
	ctx, cancel := context.WithCancel(context.Background())
	jwks, err := keyfunc.NewDefaultCtx(ctx, []string{keycloakJWKSURL})
	if err != nil {
		cancel()
		return nil, err
	}

	parser := jwt.NewParser(
		jwt.WithIssuer(keycloakIssuer),
		jwt.WithAudience(keycloakAudience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(30),
	)

	return &connectHandler{
		parser:  parser,
		keyFunc: jwks.Keyfunc,
		stopJWKS: func() error {
			cancel()
			return nil
		},
	}, nil
}

func (handler *connectHandler) handleConnect(_ context.Context, event events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	token, err := extractToken(event)
	if err != nil {
		return unauthorized("missing or malformed token"), nil
	}

	claims, err := handler.validateToken(token)
	if err != nil {
		return unauthorized("invalid token"), nil
	}

	connectionID := event.RequestContext.ConnectionID
	log.Printf("accepted websocket connection id=%s sub=%s preferred_username=%s", connectionID, claims.Subject, claims.PreferredUsername)

	// Persist connectionId + claims here if your application needs fan-out, presence, or authorization checks later.
	// Example storage choices: DynamoDB, Redis, Aurora, or another low-latency lookup store.

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       "connected",
	}, nil
}

func (handler *connectHandler) validateToken(rawToken string) (*websocketClaims, error) {
	claims := &websocketClaims{}
	parsedToken, err := handler.parser.ParseWithClaims(rawToken, claims, handler.keyFunc)
	if err != nil {
		return nil, err
	}
	if !parsedToken.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

func extractToken(event events.APIGatewayWebsocketProxyRequest) (string, error) {
	if token := bearerToken(event.Headers["Authorization"]); token != "" {
		return token, nil
	}
	if token := bearerToken(event.Headers["authorization"]); token != "" {
		return token, nil
	}
	if token := strings.TrimSpace(event.QueryStringParameters["token"]); token != "" {
		return token, nil
	}
	if token := subprotocolToken(event.Headers["Sec-WebSocket-Protocol"]); token != "" {
		return token, nil
	}
	if token := subprotocolToken(event.Headers["sec-websocket-protocol"]); token != "" {
		return token, nil
	}

	return "", errors.New("token not found")
}

func bearerToken(headerValue string) string {
	headerValue = strings.TrimSpace(headerValue)
	if headerValue == "" {
		return ""
	}

	parts := strings.Fields(headerValue)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	if len(parts) == 1 && strings.Count(parts[0], ".") == 2 {
		return parts[0]
	}

	return ""
}

func subprotocolToken(headerValue string) string {
	for _, part := range strings.Split(headerValue, ",") {
		protocolValue := strings.TrimSpace(part)
		switch {
		case strings.Count(protocolValue, ".") == 2:
			return protocolValue
		case strings.HasPrefix(strings.ToLower(protocolValue), "bearer."):
			token := strings.TrimSpace(protocolValue[len("bearer."):])
			if strings.Count(token, ".") == 2 {
				return token
			}
		}
	}

	return ""
}

func unauthorized(message string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusUnauthorized,
		Body:       message,
	}
}
