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
	parser           *jwt.Parser
	keyFunc          jwt.Keyfunc
	allowedIssuers   []string
	allowedAudiences []string
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

	parser := jwt.NewParser(options...)
	keyResolver, closer, err := newKeyFunc(cfg)
	if err != nil {
		return nil, nil, err
	}

	return &JWTValidator{
		parser:           parser,
		keyFunc:          keyResolver,
		allowedIssuers:   allowedValues(cfg.Issuer, cfg.Issuers),
		allowedAudiences: allowedValues(cfg.Audience, cfg.Audiences),
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

	claims := claimsFromMapClaims(claimSet)

	if len(v.allowedIssuers) > 0 && !contains(v.allowedIssuers, claims.Issuer) {
		return nil, ErrInvalidToken
	}
	if len(v.allowedAudiences) > 0 && !containsAny(v.allowedAudiences, claims.Audience) {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// allowedValues merges a single legacy value with a plural slice into one
// deduplicated allowlist. Returns nil (no restriction) when both are empty.
func allowedValues(single string, plural []string) []string {
	if single == "" && len(plural) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(plural)+1)
	out := make([]string, 0, len(plural)+1)

	add := func(value string) {
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}

	add(single)
	for _, value := range plural {
		add(value)
	}

	return out
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsAny(allowed []string, candidates []string) bool {
	for _, candidate := range candidates {
		if contains(allowed, candidate) {
			return true
		}
	}
	return false
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

		var jwks keyfunc.Keyfunc
		var err error

		if cfg.JWKSRequestTimeout > 0 || cfg.JWKSRefreshInterval > 0 || cfg.JWKSRefreshErrorHandler != nil {
			override := keyfunc.Override{
				HTTPTimeout:     cfg.JWKSRequestTimeout,
				RefreshInterval: cfg.JWKSRefreshInterval,
			}
			if cfg.JWKSRefreshErrorHandler != nil {
				handler := cfg.JWKSRefreshErrorHandler
				override.RefreshErrorHandlerFunc = func(url string) func(ctx context.Context, err error) {
					return func(_ context.Context, err error) {
						handler(url, err)
					}
				}
			}
			jwks, err = keyfunc.NewDefaultOverrideCtx(ctx, []string{cfg.JWKSURL}, override)
		} else {
			jwks, err = keyfunc.NewDefaultCtx(ctx, []string{cfg.JWKSURL})
		}
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
