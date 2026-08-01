package wsauthkit

import (
	"context"
	"fmt"
	"strings"
	"sync"
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

// BatchRelationChecker is an optional RelationChecker extension for engines
// that can answer "does subject have relation to any of these resources" in
// a single call (e.g. SpiceDB BulkCheckPermission, OpenFGA batch check).
//
// Implement this when checking many resources per request is common (e.g.
// listing which rooms/workspaces a reconnecting user can see) to avoid one
// remote call per resource. If a ReBACAuthorizer's Checker does not
// implement it, AuthorizeBatch falls back to bounded-concurrency calls to
// HasRelation.
type BatchRelationChecker interface {
	HasRelations(ctx context.Context, subject, relation string, resources []string) (map[string]bool, error)
}

// BatchAuthorizer is an optional Authorizer extension for checking many
// resources against the same subject/action in one call.
type BatchAuthorizer interface {
	AuthorizeBatch(ctx context.Context, input AuthorizationInput, resources []string) (map[string]bool, error)
}

// batchFanOutLimit bounds concurrent HasRelation calls when a
// RelationChecker does not implement BatchRelationChecker.
const batchFanOutLimit = 8

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

// AuthorizeBatch evaluates the same action against many resources at once.
// RBAC policies are local, so this is a cheap loop with no remote calls.
func (a RBACAuthorizer) AuthorizeBatch(ctx context.Context, input AuthorizationInput, resources []string) (map[string]bool, error) {
	results := make(map[string]bool, len(resources))

	for _, resource := range resources {
		allowed, err := a.Authorize(ctx, AuthorizationInput{
			Claims:   input.Claims,
			Subject:  input.Subject,
			Action:   input.Action,
			Resource: resource,
		})
		if err != nil {
			return nil, err
		}

		results[resource] = allowed
	}

	return results, nil
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

// AuthorizeBatch evaluates the same relation against many resources at
// once. If Checker implements BatchRelationChecker, it is used directly
// (a single remote call). Otherwise this falls back to concurrent
// HasRelation calls, bounded by batchFanOutLimit, so checking N resources
// against a per-resource-only backend doesn't serialize N round trips.
func (a ReBACAuthorizer) AuthorizeBatch(ctx context.Context, input AuthorizationInput, resources []string) (map[string]bool, error) {
	if a.Checker == nil {
		return nil, fmt.Errorf("wsauthkit: relation checker is not configured")
	}

	subject := input.Subject
	if subject == "" && input.Claims != nil {
		subject = input.Claims.Subject
	}
	if subject == "" {
		return nil, fmt.Errorf("wsauthkit: subject is required for ReBAC authorization")
	}

	if len(resources) == 0 {
		return map[string]bool{}, nil
	}

	if batchChecker, ok := a.Checker.(BatchRelationChecker); ok {
		return batchChecker.HasRelations(ctx, subject, input.Action, resources)
	}

	return fanOutRelationChecks(ctx, a.Checker, subject, input.Action, resources)
}

func fanOutRelationChecks(ctx context.Context, checker RelationChecker, subject, relation string, resources []string) (map[string]bool, error) {
	type outcome struct {
		resource string
		allowed  bool
		err      error
	}

	jobs := make(chan string)
	results := make(chan outcome, len(resources))

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	workerCount := batchFanOutLimit
	if len(resources) < workerCount {
		workerCount = len(resources)
	}

	var wg sync.WaitGroup
	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go func() {
			defer wg.Done()
			for resource := range jobs {
				allowed, err := checker.HasRelation(ctx, subject, relation, resource)
				results <- outcome{resource: resource, allowed: allowed, err: err}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, resource := range resources {
			select {
			case jobs <- resource:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	out := make(map[string]bool, len(resources))
	for result := range results {
		if result.err != nil {
			cancel()
			return nil, result.err
		}
		out[result.resource] = result.allowed
	}

	return out, nil
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

// AuthorizeBatch evaluates every resource against each child authorizer in
// order, short-circuiting a resource as soon as one child allows it (OR
// semantics, matching Authorize). Children implementing BatchAuthorizer are
// queried once for all still-undecided resources; others fall back to
// Authorize per resource.
func (a AnyAuthorizer) AuthorizeBatch(ctx context.Context, input AuthorizationInput, resources []string) (map[string]bool, error) {
	results := make(map[string]bool, len(resources))
	remaining := make([]string, len(resources))
	copy(remaining, resources)

	for _, authorizer := range a {
		if authorizer == nil || len(remaining) == 0 {
			continue
		}

		var partial map[string]bool
		var err error

		if batchAuthorizer, ok := authorizer.(BatchAuthorizer); ok {
			partial, err = batchAuthorizer.AuthorizeBatch(ctx, input, remaining)
		} else {
			partial = make(map[string]bool, len(remaining))
			for _, resource := range remaining {
				var allowed bool
				allowed, err = authorizer.Authorize(ctx, AuthorizationInput{
					Claims:   input.Claims,
					Subject:  input.Subject,
					Action:   input.Action,
					Resource: resource,
				})
				if err != nil {
					break
				}
				partial[resource] = allowed
			}
		}
		if err != nil {
			return nil, err
		}

		stillRemaining := remaining[:0]
		for _, resource := range remaining {
			if partial[resource] {
				results[resource] = true
			} else {
				stillRemaining = append(stillRemaining, resource)
			}
		}
		remaining = stillRemaining
	}

	for _, resource := range remaining {
		results[resource] = false
	}

	return results, nil
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
