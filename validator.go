package wsauthkit

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

type jsonNumber = json.Number

// KeyFunc resolves the verification key for a JWT token.
type KeyFunc = jwt.Keyfunc

// TokenValidator validates a token and returns normalized claims.
type TokenValidator interface {
	ValidateToken(token string) (*Claims, error)
}

// JWTValidator validates JWT tokens using a signing key, KeyFunc or JWKS.
type JWTValidator struct {
	parser  *jwt.Parser
	keyFunc jwt.Keyfunc
}

// NewJWTValidator builds a JWTValidator and optional closer from Config.
func NewJWTValidator(cfg Config) (TokenValidator, func() error, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, nil, err
	}

	options := []jwt.ParserOption{
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(30 * time.Second),
	}

	if cfg.Issuer != "" {
		options = append(options, jwt.WithIssuer(cfg.Issuer))
	}
	if cfg.Audience != "" {
		options = append(options, jwt.WithAudience(cfg.Audience))
	}

	parser := jwt.NewParser(options...)
	keyResolver, closer, err := newKeyFunc(cfg)
	if err != nil {
		return nil, nil, err
	}

	return &JWTValidator{
		parser:  parser,
		keyFunc: keyResolver,
	}, closer, nil
}

// ValidateToken parses, verifies and normalizes a JWT.
func (v *JWTValidator) ValidateToken(token string) (*Claims, error) {
	claimSet := jwt.MapClaims{}
	parsedToken, err := v.parser.ParseWithClaims(token, claimSet, v.keyFunc)
	if err != nil {
		return nil, errors.Join(ErrInvalidToken, err)
	}
	if !parsedToken.Valid {
		return nil, ErrInvalidToken
	}

	return claimsFromMapClaims(claimSet), nil
}

func newKeyFunc(cfg Config) (jwt.Keyfunc, func() error, error) {
	switch {
	case cfg.KeyFunc != nil:
		return cfg.KeyFunc, nil, nil
	case cfg.SigningKey != nil:
		return func(_ *jwt.Token) (any, error) {
			return cfg.SigningKey, nil
		}, nil, nil
	default:
		ctx, cancel := context.WithCancel(context.Background())
		jwks, err := keyfunc.NewDefaultCtx(ctx, []string{cfg.JWKSURL})
		if err != nil {
			cancel()
			return nil, nil, err
		}

		return jwks.Keyfunc, func() error {
			cancel()
			return nil
		}, nil
	}
}
