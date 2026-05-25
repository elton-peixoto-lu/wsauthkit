package wsauthkit

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

type appIdentity struct {
	UserID   string
	TenantID string
	Roles    []string
	Verified bool
}

func TestMapClaimsMapsTypedStruct(t *testing.T) {
	t.Parallel()

	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "user-123"},
		Values: map[string]any{
			"tenant_id":      "tenant-1",
			"roles":          []any{"admin", "editor"},
			"email_verified": true,
		},
	}

	identity, err := MapClaims(claims, func(r ClaimReader) (appIdentity, error) {
		tenantID, err := r.RequiredString("tenant_id")
		if err != nil {
			return appIdentity{}, err
		}
		roles, _ := r.Strings("roles")
		verified, _ := r.Bool("email_verified")

		return appIdentity{
			UserID:   r.Subject(),
			TenantID: tenantID,
			Roles:    roles,
			Verified: verified,
		}, nil
	})
	if err != nil {
		t.Fatalf("map claims: %v", err)
	}

	if identity.UserID != "user-123" {
		t.Fatalf("unexpected user id: %q", identity.UserID)
	}
	if identity.TenantID != "tenant-1" {
		t.Fatalf("unexpected tenant id: %q", identity.TenantID)
	}
	if len(identity.Roles) != 2 || identity.Roles[0] != "admin" {
		t.Fatalf("unexpected roles: %#v", identity.Roles)
	}
	if !identity.Verified {
		t.Fatal("expected verified=true")
	}
}

func TestMapClaimsReturnsErrorForMissingRequiredClaim(t *testing.T) {
	t.Parallel()

	_, err := MapClaims(&Claims{Values: map[string]any{}}, func(r ClaimReader) (appIdentity, error) {
		tenantID, err := r.RequiredString("tenant_id")
		if err != nil {
			return appIdentity{}, err
		}
		return appIdentity{TenantID: tenantID}, nil
	})
	if err == nil {
		t.Fatal("expected error for missing required claim")
	}
}

func TestMapClaimsRejectsNilInputs(t *testing.T) {
	t.Parallel()

	_, err := MapClaims[appIdentity](nil, func(ClaimReader) (appIdentity, error) { return appIdentity{}, nil })
	if err == nil {
		t.Fatal("expected nil claims error")
	}

	_, err = MapClaims[appIdentity](&Claims{Values: map[string]any{}}, nil)
	if err == nil {
		t.Fatal("expected nil mapper error")
	}
}

