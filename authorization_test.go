package wsauthkit

import (
	"context"
	"errors"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

type mockRelationChecker struct {
	allowed bool
	err     error
}

func (m mockRelationChecker) HasRelation(_ context.Context, _ string, _ string, _ string) (bool, error) {
	return m.allowed, m.err
}

func TestRBACAuthorizerAuthorize(t *testing.T) {
	authorizer := RBACAuthorizer{
		Policies: map[string][]string{
			"admin": {"*:*"},
			"user":  {"read:doc/1"},
		},
	}

	claims := &Claims{Values: map[string]any{"roles": []string{"user"}}}
	allowed, err := authorizer.Authorize(context.Background(), AuthorizationInput{
		Claims:   claims,
		Action:   "read",
		Resource: "doc/1",
	})
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if !allowed {
		t.Fatal("expected permission to be allowed")
	}
}

func TestRBACAuthorizerDenyWhenNoMatch(t *testing.T) {
	authorizer := RBACAuthorizer{Policies: map[string][]string{"user": {"read:doc/1"}}}
	claims := &Claims{Values: map[string]any{"roles": []string{"user"}}}

	allowed, err := authorizer.Authorize(context.Background(), AuthorizationInput{
		Claims:   claims,
		Action:   "write",
		Resource: "doc/1",
	})
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if allowed {
		t.Fatal("expected permission to be denied")
	}
}

func TestRBACAuthorizerUsesCustomRoleClaim(t *testing.T) {
	authorizer := RBACAuthorizer{
		RoleClaim: "groups",
		Policies:  map[string][]string{"ops": {"deploy:service/a"}},
	}
	claims := &Claims{Values: map[string]any{"groups": []any{"ops"}}}

	allowed, err := authorizer.Authorize(context.Background(), AuthorizationInput{
		Claims:   claims,
		Action:   "deploy",
		Resource: "service/a",
	})
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if !allowed {
		t.Fatal("expected permission to be allowed from custom role claim")
	}
}

func TestReBACAuthorizerAuthorize(t *testing.T) {
	authorizer := ReBACAuthorizer{Checker: mockRelationChecker{allowed: true}}
	allowed, err := authorizer.Authorize(context.Background(), AuthorizationInput{
		Subject:  "user:1",
		Action:   "viewer",
		Resource: "workspace:1",
	})
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if !allowed {
		t.Fatal("expected relation to be allowed")
	}
}

func TestReBACAuthorizerUsesClaimsSubject(t *testing.T) {
	authorizer := ReBACAuthorizer{Checker: mockRelationChecker{allowed: true}}
	allowed, err := authorizer.Authorize(context.Background(), AuthorizationInput{
		Claims:   &Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "user:claims"}},
		Action:   "viewer",
		Resource: "workspace:1",
	})
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if !allowed {
		t.Fatal("expected relation to be allowed from claims subject")
	}
}

func TestReBACAuthorizerErrorFromChecker(t *testing.T) {
	expected := errors.New("backend unavailable")
	authorizer := ReBACAuthorizer{Checker: mockRelationChecker{err: expected}}

	_, err := authorizer.Authorize(context.Background(), AuthorizationInput{
		Subject:  "user:1",
		Action:   "viewer",
		Resource: "workspace:1",
	})
	if !errors.Is(err, expected) {
		t.Fatalf("expected propagated error, got %v", err)
	}
}

func TestReBACAuthorizerRequiresChecker(t *testing.T) {
	authorizer := ReBACAuthorizer{}
	_, err := authorizer.Authorize(context.Background(), AuthorizationInput{
		Subject:  "user:1",
		Action:   "viewer",
		Resource: "workspace:1",
	})
	if err == nil {
		t.Fatal("expected checker configuration error")
	}
}

func TestAnyAuthorizerAuthorize(t *testing.T) {
	authorizer := AnyAuthorizer{
		AuthorizationFunc(func(context.Context, AuthorizationInput) (bool, error) { return false, nil }),
		AuthorizationFunc(func(context.Context, AuthorizationInput) (bool, error) { return true, nil }),
	}

	allowed, err := authorizer.Authorize(context.Background(), AuthorizationInput{})
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if !allowed {
		t.Fatal("expected composed authorizer to allow")
	}
}
