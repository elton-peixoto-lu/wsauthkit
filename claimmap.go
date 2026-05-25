package wsauthkit

import (
	"fmt"
	"strings"
)

// ClaimReader provides lightweight typed access to validated claims.
// This layer is optional and can be used when applications want typed mapping.
type ClaimReader struct {
	claims *Claims
}

// NewClaimReader builds a typed claim reader for a validated claim set.
func NewClaimReader(claims *Claims) ClaimReader {
	return ClaimReader{claims: claims}
}

// MapClaims maps validated claims into an application-specific type.
// The mapper callback keeps the API explicit and avoids reflection-heavy decoding.
func MapClaims[T any](claims *Claims, mapper func(ClaimReader) (T, error)) (T, error) {
	var zero T
	if claims == nil {
		return zero, fmt.Errorf("wsauthkit: claims are required")
	}
	if mapper == nil {
		return zero, fmt.Errorf("wsauthkit: mapper function is required")
	}

	return mapper(NewClaimReader(claims))
}

// Subject returns the JWT subject (sub).
func (r ClaimReader) Subject() string {
	if r.claims == nil {
		return ""
	}
	return r.claims.Subject
}

// Issuer returns the JWT issuer (iss).
func (r ClaimReader) Issuer() string {
	if r.claims == nil {
		return ""
	}
	return r.claims.Issuer
}

// Audience returns the JWT audience (aud).
func (r ClaimReader) Audience() []string {
	if r.claims == nil || len(r.claims.Audience) == 0 {
		return nil
	}
	return []string(r.claims.Audience)
}

// String returns a private claim as string.
func (r ClaimReader) String(name string) (string, bool) {
	value, ok := r.raw(name)
	if !ok {
		return "", false
	}
	text, ok := value.(string)
	if !ok || text == "" {
		return "", false
	}
	return text, true
}

// RequiredString returns a private claim as string or error when missing/invalid.
func (r ClaimReader) RequiredString(name string) (string, error) {
	value, ok := r.String(name)
	if !ok {
		return "", fmt.Errorf("wsauthkit: required string claim %q is missing or invalid", name)
	}
	return value, nil
}

// Strings returns a private claim as []string.
// Supported formats: string, []string, []any of strings.
func (r ClaimReader) Strings(name string) ([]string, bool) {
	value, ok := r.raw(name)
	if !ok {
		return nil, false
	}

	switch typed := value.(type) {
	case string:
		if typed == "" {
			return nil, false
		}
		return []string{typed}, true
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out, len(out) > 0
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil, false
			}
			if trimmed := strings.TrimSpace(text); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out, len(out) > 0
	default:
		return nil, false
	}
}

// Bool returns a private claim as bool.
func (r ClaimReader) Bool(name string) (bool, bool) {
	value, ok := r.raw(name)
	if !ok {
		return false, false
	}
	flag, ok := value.(bool)
	return flag, ok
}

// Int64 returns a private claim as int64.
func (r ClaimReader) Int64(name string) (int64, bool) {
	value, ok := r.raw(name)
	if !ok {
		return 0, false
	}

	switch typed := value.(type) {
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	case float64:
		return int64(typed), true
	case jsonNumber:
		asInt, err := typed.Int64()
		if err != nil {
			return 0, false
		}
		return asInt, true
	default:
		return 0, false
	}
}

func (r ClaimReader) raw(name string) (any, bool) {
	if r.claims == nil {
		return nil, false
	}
	return r.claims.Value(name)
}
