package wsauthkit

import (
	"errors"
	"net/http"
)

// Config defines the public setup surface for WSAuthKit.
type Config struct {
	Issuer       string
	Audience     string
	JWKSURL      string
	SigningKey   any
	KeyFunc      KeyFunc
	Extractors   []TokenExtractor
	ErrorHandler ErrorHandler
}

// Auth wires token extraction, token validation and context injection.
type Auth struct {
	extractor    TokenExtractor
	validator    TokenValidator
	errorHandler ErrorHandler
	closer       func() error
}

// NewAuth constructs an Auth middleware with secure defaults.
func NewAuth(cfg Config) (*Auth, error) {
	extractor := ChainExtractors(cfg.Extractors...)
	if extractor == nil {
		extractor = DefaultExtractor()
	}

	validator, closer, err := NewJWTValidator(cfg)
	if err != nil {
		return nil, err
	}

	handler := cfg.ErrorHandler
	if handler == nil {
		handler = DefaultErrorHandler
	}

	return &Auth{
		extractor:    extractor,
		validator:    validator,
		errorHandler: handler,
		closer:       closer,
	}, nil
}

// ExtractToken exposes the configured extraction pipeline for standalone usage.
func (a *Auth) ExtractToken(r *http.Request) (string, error) {
	if a == nil || a.extractor == nil {
		return "", ErrExtractorNotConfigured
	}

	return a.extractor.ExtractToken(r)
}

// ValidateToken exposes the configured validator for standalone usage.
func (a *Auth) ValidateToken(token string) (*Claims, error) {
	if a == nil || a.validator == nil {
		return nil, ErrValidatorNotConfigured
	}

	return a.validator.ValidateToken(token)
}

// Authenticate runs the full auth pipeline and returns claims on success.
func (a *Auth) Authenticate(r *http.Request) (*Claims, error) {
	if a == nil {
		return nil, ErrAuthNotConfigured
	}

	token, err := a.ExtractToken(r)
	if err != nil {
		return nil, err
	}

	claims, err := a.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	return claims, nil
}

// Close releases internal background resources such as JWKS refreshers.
func (a *Auth) Close() error {
	if a == nil || a.closer == nil {
		return nil
	}

	return a.closer()
}

func validateConfig(cfg Config) error {
	switch {
	case cfg.KeyFunc == nil && cfg.SigningKey == nil && cfg.JWKSURL == "":
		return ErrMissingKeySource
	case cfg.KeyFunc != nil && (cfg.SigningKey != nil || cfg.JWKSURL != ""):
		return errors.New("wsauthkit: key func cannot be combined with signing key or JWKS URL")
	case cfg.SigningKey != nil && cfg.JWKSURL != "":
		return errors.New("wsauthkit: signing key cannot be combined with JWKS URL")
	default:
		return nil
	}
}
