package wsauthkit

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims is the normalized claim container exposed to application handlers.
type Claims struct {
	jwt.RegisteredClaims
	Values map[string]any
}

// Value returns a private claim by name.
func (c *Claims) Value(name string) (any, bool) {
	if c == nil || c.Values == nil {
		return nil, false
	}

	value, ok := c.Values[name]
	return value, ok
}

// MustValue returns a private claim by name or panics if it is missing.
func (c *Claims) MustValue(name string) any {
	value, ok := c.Value(name)
	if !ok {
		panic(fmt.Sprintf("wsauthkit: missing claim %q", name))
	}

	return value
}

// AsMap returns a shallow copy of all claims, including registered claims.
func (c *Claims) AsMap() map[string]any {
	if c == nil {
		return nil
	}

	out := make(map[string]any, len(c.Values)+8)
	for key, value := range c.Values {
		out[key] = value
	}

	if c.Issuer != "" {
		out["iss"] = c.Issuer
	}
	if c.Subject != "" {
		out["sub"] = c.Subject
	}
	if len(c.Audience) > 0 {
		out["aud"] = []string(c.Audience)
	}
	if c.ID != "" {
		out["jti"] = c.ID
	}
	if c.ExpiresAt != nil {
		out["exp"] = c.ExpiresAt.Time.Unix()
	}
	if c.NotBefore != nil {
		out["nbf"] = c.NotBefore.Time.Unix()
	}
	if c.IssuedAt != nil {
		out["iat"] = c.IssuedAt.Time.Unix()
	}

	return out
}

func claimsFromMapClaims(claimSet jwt.MapClaims) *Claims {
	claims := &Claims{
		Values: make(map[string]any, len(claimSet)),
	}

	for key, value := range claimSet {
		switch key {
		case "iss":
			claims.Issuer = stringClaim(value)
		case "sub":
			claims.Subject = stringClaim(value)
		case "aud":
			claims.Audience = audienceClaim(value)
		case "jti":
			claims.ID = stringClaim(value)
		case "exp":
			claims.ExpiresAt = numericDateClaim(value)
		case "nbf":
			claims.NotBefore = numericDateClaim(value)
		case "iat":
			claims.IssuedAt = numericDateClaim(value)
		default:
			claims.Values[key] = value
		}
	}

	return claims
}

func stringClaim(value any) string {
	text, _ := value.(string)
	return text
}

func audienceClaim(value any) jwt.ClaimStrings {
	switch typed := value.(type) {
	case string:
		if typed == "" {
			return nil
		}
		return jwt.ClaimStrings{typed}
	case []string:
		return jwt.ClaimStrings(typed)
	case []any:
		audienceValues := make(jwt.ClaimStrings, 0, len(typed))
		for _, audienceValue := range typed {
			if text, ok := audienceValue.(string); ok && text != "" {
				audienceValues = append(audienceValues, text)
			}
		}
		return audienceValues
	default:
		return nil
	}
}

func numericDateClaim(value any) *jwt.NumericDate {
	switch typed := value.(type) {
	case float64:
		return jwt.NewNumericDate(time.Unix(int64(typed), 0))
	case int64:
		return jwt.NewNumericDate(time.Unix(typed, 0))
	case int:
		return jwt.NewNumericDate(time.Unix(int64(typed), 0))
	case jsonNumber:
		if asInt, err := typed.Int64(); err == nil {
			return jwt.NewNumericDate(time.Unix(asInt, 0))
		}
	}

	return nil
}
