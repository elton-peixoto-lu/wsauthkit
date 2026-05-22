package wsauthkit

import (
	"context"
	"fmt"
	"strings"
)

// AuthorizationInput is the canonical input passed to authorization engines.
type AuthorizationInput struct {
	Claims   *Claims
	Subject  string
	Action   string
	Resource string
}

// Authorizer decides whether a given action over a resource is allowed.
type Authorizer interface {
	Authorize(ctx context.Context, input AuthorizationInput) (bool, error)
}

// AuthorizationFunc adapts plain functions to the Authorizer interface.
type AuthorizationFunc func(ctx context.Context, input AuthorizationInput) (bool, error)

// Authorize calls f(ctx, input).
func (f AuthorizationFunc) Authorize(ctx context.Context, input AuthorizationInput) (bool, error) {
	return f(ctx, input)
}

// RelationChecker is the minimal ReBAC extension point.
// It can be backed by SpiceDB/OpenFGA or any in-house engine.
type RelationChecker interface {
	HasRelation(ctx context.Context, subject, relation, resource string) (bool, error)
}

// RBACAuthorizer evaluates action-resource permissions from roles.
//
// Permission format: "<action>:<resource>", wildcard supported for either side
// (e.g. "read:*", "*:workspace/123", or "*:*").
type RBACAuthorizer struct {
	RoleClaim string
	Policies  map[string][]string
}

// Authorize evaluates RBAC permission checks against claims.
func (a RBACAuthorizer) Authorize(_ context.Context, input AuthorizationInput) (bool, error) {
	if input.Claims == nil {
		return false, fmt.Errorf("wsauthkit: claims are required for RBAC authorization")
	}

	roles := extractClaimStrings(input.Claims, a.RoleClaim)
	if len(roles) == 0 {
		return false, nil
	}

	required := permission(input.Action, input.Resource)
	for _, role := range roles {
		for _, granted := range a.Policies[role] {
			if permissionMatches(granted, required) {
				return true, nil
			}
		}
	}

	return false, nil
}

// ReBACAuthorizer delegates relation checks to a pluggable relation engine.
// By convention, action is treated as the relation name.
type ReBACAuthorizer struct {
	Checker RelationChecker
}

// Authorize evaluates relation-based permission checks.
func (a ReBACAuthorizer) Authorize(ctx context.Context, input AuthorizationInput) (bool, error) {
	if a.Checker == nil {
		return false, fmt.Errorf("wsauthkit: relation checker is not configured")
	}

	subject := input.Subject
	if subject == "" && input.Claims != nil {
		subject = input.Claims.Subject
	}
	if subject == "" {
		return false, fmt.Errorf("wsauthkit: subject is required for ReBAC authorization")
	}

	return a.Checker.HasRelation(ctx, subject, input.Action, input.Resource)
}

// AnyAuthorizer composes multiple authorizers with OR semantics.
// It allows hybrid models such as RBAC + ReBAC.
type AnyAuthorizer []Authorizer

// Authorize returns true when any child authorizer allows the input.
func (a AnyAuthorizer) Authorize(ctx context.Context, input AuthorizationInput) (bool, error) {
	for _, authorizer := range a {
		if authorizer == nil {
			continue
		}

		allowed, err := authorizer.Authorize(ctx, input)
		if err != nil {
			return false, err
		}
		if allowed {
			return true, nil
		}
	}

	return false, nil
}

func permission(action, resource string) string {
	return strings.TrimSpace(action) + ":" + strings.TrimSpace(resource)
}

func permissionMatches(granted, required string) bool {
	gAction, gResource, ok := strings.Cut(strings.TrimSpace(granted), ":")
	if !ok {
		return false
	}
	rAction, rResource, ok := strings.Cut(strings.TrimSpace(required), ":")
	if !ok {
		return false
	}

	return (gAction == "*" || gAction == rAction) && (gResource == "*" || gResource == rResource)
}

func extractClaimStrings(claims *Claims, claimName string) []string {
	if claims == nil {
		return nil
	}
	if claimName == "" {
		claimName = "roles"
	}

	raw, ok := claims.Value(claimName)
	if !ok {
		return nil
	}

	switch typed := raw.(type) {
	case string:
		if typed == "" {
			return nil
		}
		return []string{typed}
	case []string:
		out := make([]string, 0, len(typed))
		for _, value := range typed {
			if value != "" {
				out = append(out, value)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(typed))
		for _, value := range typed {
			if text, ok := value.(string); ok && text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}
