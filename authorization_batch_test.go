package wsauthkit

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
)

func TestRBACAuthorizerAuthorizeBatch(t *testing.T) {
	t.Parallel()

	authorizer := RBACAuthorizer{
		Policies: map[string][]string{
			"user": {"read:doc/1", "read:doc/2"},
		},
	}
	claims := &Claims{Values: map[string]any{"roles": []string{"user"}}}

	results, err := authorizer.AuthorizeBatch(context.Background(), AuthorizationInput{
		Claims: claims,
		Action: "read",
	}, []string{"doc/1", "doc/2", "doc/3"})
	if err != nil {
		t.Fatalf("authorize batch: %v", err)
	}

	expected := map[string]bool{"doc/1": true, "doc/2": true, "doc/3": false}
	for resource, want := range expected {
		if results[resource] != want {
			t.Fatalf("resource %q: expected %v, got %v", resource, want, results[resource])
		}
	}
}

type countingRelationChecker struct {
	allowed map[string]bool
	calls   int32
}

func (c *countingRelationChecker) HasRelation(_ context.Context, _, _, resource string) (bool, error) {
	atomic.AddInt32(&c.calls, 1)
	return c.allowed[resource], nil
}

func TestReBACAuthorizerAuthorizeBatchFallsBackToFanOut(t *testing.T) {
	t.Parallel()

	checker := &countingRelationChecker{allowed: map[string]bool{
		"workspace:1": true,
		"workspace:3": true,
	}}
	authorizer := ReBACAuthorizer{Checker: checker}

	resources := []string{"workspace:1", "workspace:2", "workspace:3"}
	results, err := authorizer.AuthorizeBatch(context.Background(), AuthorizationInput{
		Subject: "user:1",
		Action:  "viewer",
	}, resources)
	if err != nil {
		t.Fatalf("authorize batch: %v", err)
	}

	expected := map[string]bool{"workspace:1": true, "workspace:2": false, "workspace:3": true}
	for resource, want := range expected {
		if results[resource] != want {
			t.Fatalf("resource %q: expected %v, got %v", resource, want, results[resource])
		}
	}

	if int(checker.calls) != len(resources) {
		t.Fatalf("expected %d HasRelation calls, got %d", len(resources), checker.calls)
	}
}

type batchOnlyRelationChecker struct {
	response      map[string]bool
	batchCalls    int32
	singleCalls   int32
	requestedSize int
}

func (c *batchOnlyRelationChecker) HasRelation(_ context.Context, _, _, _ string) (bool, error) {
	atomic.AddInt32(&c.singleCalls, 1)
	return false, nil
}

func (c *batchOnlyRelationChecker) HasRelations(_ context.Context, _, _ string, resources []string) (map[string]bool, error) {
	atomic.AddInt32(&c.batchCalls, 1)
	c.requestedSize = len(resources)
	return c.response, nil
}

func TestReBACAuthorizerAuthorizeBatchUsesBatchRelationChecker(t *testing.T) {
	t.Parallel()

	checker := &batchOnlyRelationChecker{response: map[string]bool{
		"room:1": true,
		"room:2": false,
	}}
	authorizer := ReBACAuthorizer{Checker: checker}

	results, err := authorizer.AuthorizeBatch(context.Background(), AuthorizationInput{
		Subject: "user:1",
		Action:  "viewer",
	}, []string{"room:1", "room:2"})
	if err != nil {
		t.Fatalf("authorize batch: %v", err)
	}

	if !results["room:1"] || results["room:2"] {
		t.Fatalf("unexpected results: %#v", results)
	}
	if checker.batchCalls != 1 {
		t.Fatalf("expected exactly 1 batch call, got %d", checker.batchCalls)
	}
	if checker.singleCalls != 0 {
		t.Fatalf("expected no per-resource calls, got %d", checker.singleCalls)
	}
	if checker.requestedSize != 2 {
		t.Fatalf("expected batch call with 2 resources, got %d", checker.requestedSize)
	}
}

func TestReBACAuthorizerAuthorizeBatchPropagatesError(t *testing.T) {
	t.Parallel()

	expected := errors.New("backend unavailable")
	authorizer := ReBACAuthorizer{Checker: mockRelationChecker{err: expected}}

	_, err := authorizer.AuthorizeBatch(context.Background(), AuthorizationInput{
		Subject: "user:1",
		Action:  "viewer",
	}, []string{"workspace:1", "workspace:2"})
	if !errors.Is(err, expected) {
		t.Fatalf("expected propagated error, got %v", err)
	}
}

func TestReBACAuthorizerAuthorizeBatchHandlesManyResources(t *testing.T) {
	t.Parallel()

	allowed := make(map[string]bool, 50)
	resources := make([]string, 0, 50)
	for i := 0; i < 50; i++ {
		resource := fmt.Sprintf("res-%d", i)
		resources = append(resources, resource)
		allowed[resource] = i%3 == 0
	}

	checker := &countingRelationChecker{allowed: allowed}
	authorizer := ReBACAuthorizer{Checker: checker}

	results, err := authorizer.AuthorizeBatch(context.Background(), AuthorizationInput{
		Subject: "user:1",
		Action:  "viewer",
	}, resources)
	if err != nil {
		t.Fatalf("authorize batch: %v", err)
	}

	for _, resource := range resources {
		if results[resource] != allowed[resource] {
			t.Fatalf("resource %q: expected %v, got %v", resource, allowed[resource], results[resource])
		}
	}
}

func TestAnyAuthorizerAuthorizeBatchORsAcrossChildren(t *testing.T) {
	t.Parallel()

	rbac := RBACAuthorizer{Policies: map[string][]string{"user": {"read:doc/1"}}}
	rebacChecker := &countingRelationChecker{allowed: map[string]bool{"doc/2": true}}
	rebac := ReBACAuthorizer{Checker: rebacChecker}

	authorizer := AnyAuthorizer{rbac, rebac}
	claims := &Claims{Values: map[string]any{"roles": []string{"user"}}}

	results, err := authorizer.AuthorizeBatch(context.Background(), AuthorizationInput{
		Claims:  claims,
		Subject: "user:1",
		Action:  "read",
	}, []string{"doc/1", "doc/2", "doc/3"})
	if err != nil {
		t.Fatalf("authorize batch: %v", err)
	}

	expected := map[string]bool{"doc/1": true, "doc/2": true, "doc/3": false}
	for resource, want := range expected {
		if results[resource] != want {
			t.Fatalf("resource %q: expected %v, got %v", resource, want, results[resource])
		}
	}

	// doc/1 was already allowed by RBAC, so ReBAC should only have been
	// asked about doc/2 and doc/3.
	if rebacChecker.calls != 2 {
		t.Fatalf("expected ReBAC to be queried for 2 remaining resources, got %d", rebacChecker.calls)
	}
}
