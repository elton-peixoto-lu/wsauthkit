package apigateway

import (
	"errors"
	"net/http"

	"github.com/aws/aws-lambda-go/events"

	"github.com/elton-peixoto-lu/wsauthkit"
)

// UnauthorizedResponse builds a safe connect response for authentication failures.
func UnauthorizedResponse(err error) events.APIGatewayProxyResponse {
	message := "unauthorized"

	switch {
	case errors.Is(err, wsauthkit.ErrTokenMissing):
		message = "authentication token missing"
	case errors.Is(err, wsauthkit.ErrInvalidAuthorization), errors.Is(err, wsauthkit.ErrInvalidSecWebSocketHeader):
		message = "malformed authentication token"
	case errors.Is(err, wsauthkit.ErrInvalidToken):
		message = "invalid authentication token"
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusUnauthorized,
		Body:       message,
	}
}
