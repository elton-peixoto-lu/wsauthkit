package apigateway

import (
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/golang-jwt/jwt/v5"

	"github.com/elton-peixoto-lu/wsauthkit"
)

func TestAuthenticateWithAuthorizationHeader(t *testing.T) {
	t.Parallel()

	auth := newGatewayAuth(t)
	token := signGatewayToken(t, []byte("secret"), jwt.MapClaims{
		"iss": "https://issuer.example.com",
		"aud": "gateway",
		"sub": "gateway-user",
		"exp": time.Now().Add(5 * time.Minute).Unix(),
		"iat": time.Now().Add(-1 * time.Minute).Unix(),
	})

	claims, err := auth.Authenticate(events.APIGatewayWebsocketProxyRequest{
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.Subject != "gateway-user" {
		t.Fatalf("unexpected subject: %s", claims.Subject)
	}
}

func TestAuthenticateWithQueryStringToken(t *testing.T) {
	t.Parallel()

	auth := newGatewayAuth(t)
	token := signGatewayToken(t, []byte("secret"), jwt.MapClaims{
		"iss": "https://issuer.example.com",
		"aud": "gateway",
		"sub": "query-user",
		"exp": time.Now().Add(5 * time.Minute).Unix(),
		"iat": time.Now().Add(-1 * time.Minute).Unix(),
	})

	claims, err := auth.Authenticate(events.APIGatewayWebsocketProxyRequest{
		QueryStringParameters: map[string]string{
			"token": token,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.Subject != "query-user" {
		t.Fatalf("unexpected subject: %s", claims.Subject)
	}
}

func TestAuthenticateWithSecWebSocketProtocol(t *testing.T) {
	t.Parallel()

	auth := newGatewayAuth(t)
	token := signGatewayToken(t, []byte("secret"), jwt.MapClaims{
		"iss": "https://issuer.example.com",
		"aud": "gateway",
		"sub": "protocol-user",
		"exp": time.Now().Add(5 * time.Minute).Unix(),
		"iat": time.Now().Add(-1 * time.Minute).Unix(),
	})

	claims, err := auth.Authenticate(events.APIGatewayWebsocketProxyRequest{
		Headers: map[string]string{
			"Sec-WebSocket-Protocol": "graphql-ws, bearer, " + token,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.Subject != "protocol-user" {
		t.Fatalf("unexpected subject: %s", claims.Subject)
	}
}

func TestAuthenticateRejectsInvalidToken(t *testing.T) {
	t.Parallel()

	auth := newGatewayAuth(t)
	token := signGatewayToken(t, []byte("wrong-secret"), jwt.MapClaims{
		"iss": "https://issuer.example.com",
		"aud": "gateway",
		"sub": "bad-user",
		"exp": time.Now().Add(5 * time.Minute).Unix(),
		"iat": time.Now().Add(-1 * time.Minute).Unix(),
	})

	_, err := auth.Authenticate(events.APIGatewayWebsocketProxyRequest{
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
	})
	if !errors.Is(err, wsauthkit.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestUnauthorizedResponseMapsErrorMessages(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		err          error
		expectedBody string
	}{
		{name: "missing", err: wsauthkit.ErrTokenMissing, expectedBody: "authentication token missing"},
		{name: "invalid", err: wsauthkit.ErrInvalidToken, expectedBody: "invalid authentication token"},
		{name: "malformed", err: wsauthkit.ErrInvalidAuthorization, expectedBody: "malformed authentication token"},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			response := UnauthorizedResponse(testCase.err)
			if response.Body != testCase.expectedBody {
				t.Fatalf("expected body %q, got %q", testCase.expectedBody, response.Body)
			}
		})
	}
}

func newGatewayAuth(t *testing.T) *Auth {
	t.Helper()

	auth, err := NewAuth(Config{
		Issuer:     "https://issuer.example.com",
		Audience:   "gateway",
		SigningKey: []byte("secret"),
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

func signGatewayToken(t *testing.T, key []byte, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}
