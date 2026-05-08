package apigateway

import (
	"strings"

	"github.com/aws/aws-lambda-go/events"

	"github.com/elton-peixoto-lu/wsauthkit"
)

// Config defines the AWS API Gateway WebSocket auth adapter configuration.
type Config struct {
	Issuer              string
	Audience            string
	JWKSURL             string
	SigningKey          any
	KeyFunc             wsauthkit.KeyFunc
	QueryParameterNames []string
}

// Auth authenticates API Gateway WebSocket connect events.
type Auth struct {
	validator           wsauthkit.TokenValidator
	closer              func() error
	queryParameterNames []string
}

// NewAuth constructs an API Gateway auth adapter backed by WSAuthKit validation.
func NewAuth(cfg Config) (*Auth, error) {
	validator, closer, err := wsauthkit.NewJWTValidator(wsauthkit.Config{
		Issuer:     cfg.Issuer,
		Audience:   cfg.Audience,
		JWKSURL:    cfg.JWKSURL,
		SigningKey: cfg.SigningKey,
		KeyFunc:    cfg.KeyFunc,
	})
	if err != nil {
		return nil, err
	}

	queryParameterNames := cfg.QueryParameterNames
	if len(queryParameterNames) == 0 {
		queryParameterNames = []string{"token", "access_token"}
	}

	return &Auth{
		validator:           validator,
		closer:              closer,
		queryParameterNames: queryParameterNames,
	}, nil
}

// Authenticate extracts and validates a token from the API Gateway WebSocket connect event.
func (auth *Auth) Authenticate(event events.APIGatewayWebsocketProxyRequest) (*wsauthkit.Claims, error) {
	if auth == nil || auth.validator == nil {
		return nil, wsauthkit.ErrValidatorNotConfigured
	}

	token, err := auth.ExtractToken(event)
	if err != nil {
		return nil, err
	}

	return auth.validator.ValidateToken(token)
}

// ExtractToken reads a token from Authorization, query string, or Sec-WebSocket-Protocol.
func (auth *Auth) ExtractToken(event events.APIGatewayWebsocketProxyRequest) (string, error) {
	if token := bearerToken(event.Headers["Authorization"]); token != "" {
		return token, nil
	}
	if token := bearerToken(event.Headers["authorization"]); token != "" {
		return token, nil
	}

	for _, queryParameterName := range auth.queryParameterNames {
		if token := strings.TrimSpace(event.QueryStringParameters[queryParameterName]); token != "" {
			return token, nil
		}
	}

	if token := subprotocolToken(event.Headers["Sec-WebSocket-Protocol"]); token != "" {
		return token, nil
	}
	if token := subprotocolToken(event.Headers["sec-websocket-protocol"]); token != "" {
		return token, nil
	}

	return "", wsauthkit.ErrTokenMissing
}

// Close releases internal background resources such as JWKS refreshers.
func (auth *Auth) Close() error {
	if auth == nil || auth.closer == nil {
		return nil
	}

	return auth.closer()
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
		case strings.HasPrefix(strings.ToLower(protocolValue), "jwt."):
			token := strings.TrimSpace(protocolValue[len("jwt."):])
			if strings.Count(token, ".") == 2 {
				return token
			}
		}
	}

	return ""
}
