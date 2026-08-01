package wsauthkit

import (
	"errors"
	"net/http"
	"time"
)

// Config defines the public setup surface for WSAuthKit.
type Config struct {
	// Issuer and Audience configure a single expected issuer/audience.
	Issuer   string
	Audience string
	// Issuers and Audiences additionally allow any of several issuers or
	// audiences (e.g. multiple IdPs in a multi-tenant deployment). They are
	// merged with Issuer/Audience rather than replacing them.
	Issuers   []string
	Audiences []string

	JWKSURL    string
	SigningKey any
	KeyFunc    KeyFunc

	// JWKSRequestTimeout bounds each JWKS HTTP fetch. Defaults to the
	// underlying JWKS client's default (currently one minute) when zero.
	JWKSRequestTimeout time.Duration
	// JWKSRefreshInterval controls how often the JWKS is re-fetched in the
	// background. Defaults to the underlying JWKS client's default
	// (currently one hour) when zero.
	JWKSRefreshInterval time.Duration
	// JWKSRefreshErrorHandler is called with the JWKS URL and error
	// whenever a background JWKS refresh fails. Defaults to logging via
	// slog when nil.
	JWKSRefreshErrorHandler func(url string, err error)

	Extractors      []TokenExtractor
	ErrorHandler    ErrorHandler
	OriginValidator OriginValidator
	Revoker         Revoker

	// OnAuthResult is called after every Authenticate call, successful or
	// not, with the resulting claims (nil on failure) and error (nil on
	// success). Use it to wire in metrics/observability without coupling
	// the library to a specific backend (Prometheus, OpenTelemetry, ...).
	OnAuthResult func(r *http.Request, claims *Claims, err error)
}

// Auth wires token extraction, token validation and context injection.
type Auth struct {
	extractor       TokenExtractor
	validator       TokenValidator
	errorHandler    ErrorHandler
	originValidator OriginValidator
	revoker         Revoker
	onAuthResult    func(r *http.Request, claims *Claims, err error)
	closer          func() error
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
		extractor:       extractor,
		validator:       validator,
		errorHandler:    handler,
		originValidator: cfg.OriginValidator,
		revoker:         cfg.Revoker,
		onAuthResult:    cfg.OnAuthResult,
		closer:          closer,
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
	claims, err := a.authenticate(r)

	if a != nil && a.onAuthResult != nil {
		a.onAuthResult(r, claims, err)
	}

	return claims, err
}

func (a *Auth) authenticate(r *http.Request) (*Claims, error) {
	if a == nil {
		return nil, ErrAuthNotConfigured
	}

	if a.originValidator != nil {
		if err := a.originValidator.ValidateOrigin(r); err != nil {
			return nil, err
		}
	}

	token, err := a.ExtractToken(r)
	if err != nil {
		return nil, err
	}

	claims, err := a.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	if a.revoker != nil {
		revoked, err := a.revoker.IsRevoked(r.Context(), claims)
		if err != nil {
			return nil, err
		}
		if revoked {
			return nil, ErrTokenRevoked
		}
	}

	return claims, nil
}

// HandleError writes the authentication failure response using the
// configured ErrorHandler (or DefaultErrorHandler). It lets framework
// adapters (Gin, Echo, Fiber, ...) reuse the same error formatting as
// Middleware without duplicating the fallback logic.
func (a *Auth) HandleError(w http.ResponseWriter, r *http.Request, err error) {
	handler := DefaultErrorHandler
	if a != nil && a.errorHandler != nil {
		handler = a.errorHandler
	}

	handler(w, r, err)
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
