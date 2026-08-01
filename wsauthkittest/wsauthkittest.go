// Package wsauthkittest provides test doubles for applications that build
// on top of wsauthkit, so consumers don't need to hand-roll fake validators
// or claim builders for their own handler tests.
package wsauthkittest

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/elton-peixoto-lu/wsauthkit"
)

// ClaimsBuilder incrementally builds a *wsauthkit.Claims for tests.
type ClaimsBuilder struct {
	claims wsauthkit.Claims
}

// NewClaims starts a ClaimsBuilder with a subject and a 5-minute expiry.
func NewClaims(subject string) *ClaimsBuilder {
	now := time.Now()
	return &ClaimsBuilder{
		claims: wsauthkit.Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   subject,
				IssuedAt:  jwt.NewNumericDate(now),
				ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
			},
			Values: map[string]any{},
		},
	}
}

// Issuer sets the issuer claim.
func (b *ClaimsBuilder) Issuer(issuer string) *ClaimsBuilder {
	b.claims.Issuer = issuer
	return b
}

// Audience sets the audience claim.
func (b *ClaimsBuilder) Audience(audience ...string) *ClaimsBuilder {
	b.claims.Audience = audience
	return b
}

// ExpiresAt overrides the expiry claim.
func (b *ClaimsBuilder) ExpiresAt(expiresAt time.Time) *ClaimsBuilder {
	b.claims.ExpiresAt = jwt.NewNumericDate(expiresAt)
	return b
}

// With sets an arbitrary private claim.
func (b *ClaimsBuilder) With(name string, value any) *ClaimsBuilder {
	b.claims.Values[name] = value
	return b
}

// Roles is a convenience for With("roles", roles).
func (b *ClaimsBuilder) Roles(roles ...string) *ClaimsBuilder {
	return b.With("roles", roles)
}

// Build returns the constructed claims.
func (b *ClaimsBuilder) Build() *wsauthkit.Claims {
	claims := b.claims
	return &claims
}

// FakeValidator is a wsauthkit.TokenValidator test double. Register tokens
// with Allow, or set Err to force every call to fail.
type FakeValidator struct {
	tokens map[string]*wsauthkit.Claims
	Err    error
}

// NewFakeValidator returns an empty FakeValidator.
func NewFakeValidator() *FakeValidator {
	return &FakeValidator{tokens: map[string]*wsauthkit.Claims{}}
}

// Allow registers claims to be returned when token is validated.
func (v *FakeValidator) Allow(token string, claims *wsauthkit.Claims) *FakeValidator {
	v.tokens[token] = claims
	return v
}

// ValidateToken implements wsauthkit.TokenValidator.
func (v *FakeValidator) ValidateToken(token string) (*wsauthkit.Claims, error) {
	if v.Err != nil {
		return nil, v.Err
	}

	claims, ok := v.tokens[token]
	if !ok {
		return nil, wsauthkit.ErrInvalidToken
	}

	return claims, nil
}

// FakeRevoker is a wsauthkit.Revoker test double keyed by subject.
type FakeRevoker struct {
	revokedSubjects map[string]struct{}
	Err             error
}

// NewFakeRevoker returns a FakeRevoker with nothing revoked.
func NewFakeRevoker() *FakeRevoker {
	return &FakeRevoker{revokedSubjects: map[string]struct{}{}}
}

// Revoke marks a subject as revoked.
func (r *FakeRevoker) Revoke(subject string) *FakeRevoker {
	r.revokedSubjects[subject] = struct{}{}
	return r
}

// IsRevoked implements wsauthkit.Revoker.
func (r *FakeRevoker) IsRevoked(_ context.Context, claims *wsauthkit.Claims) (bool, error) {
	if r.Err != nil {
		return false, r.Err
	}
	if claims == nil {
		return false, nil
	}

	_, revoked := r.revokedSubjects[claims.Subject]
	return revoked, nil
}
