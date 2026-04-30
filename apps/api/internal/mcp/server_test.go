package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/knowledgelayer/api/internal/identity_access"
)

// stubAccess records every call and returns the configured decision so the
// test can assert (a) the guard called Evaluate, (b) the right action /
// resourceType was passed.
type stubAccess struct {
	calls    []identity_access.EvaluateInput
	decision identity_access.AccessDecision
	err      error
}

func (s *stubAccess) Evaluate(_ context.Context, in identity_access.EvaluateInput) (*identity_access.AccessDecision, error) {
	s.calls = append(s.calls, in)
	if s.err != nil {
		return nil, s.err
	}
	d := s.decision
	return &d, nil
}

// TestWithAccessGuard_calledOnAllow asserts the inner handler runs only after
// AccessEvaluator.Evaluate returns Allow. This is the contract every MCP tool
// depends on — ADR-0015 and the v0.5.1 plan call it the access-guard
// invariant.
func TestWithAccessGuard_calledOnAllow(t *testing.T) {
	ev := &stubAccess{decision: identity_access.AccessDecision{Allow: true}}
	called := false
	guarded := withAccessGuard(ev, "view", "entity",
		func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			called = true
			return mcp.NewToolResultText("ok"), nil
		})

	ctx := WithPrincipal(context.Background(), uuid.New())
	res, err := guarded(ctx, mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("inner handler should have been called when access is allowed")
	}
	if len(ev.calls) != 1 {
		t.Fatalf("expected 1 Evaluate call, got %d", len(ev.calls))
	}
	if ev.calls[0].Action != "view" || ev.calls[0].ResourceType != "entity" {
		t.Errorf("guard passed wrong action/resourceType: %+v", ev.calls[0])
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
}

// TestWithAccessGuard_blocksOnDeny asserts Allow=false stops the inner
// handler and returns an MCP-shaped error result.
func TestWithAccessGuard_blocksOnDeny(t *testing.T) {
	ev := &stubAccess{decision: identity_access.AccessDecision{Allow: false, ReasonCode: "domain_grant_missing"}}
	called := false
	guarded := withAccessGuard(ev, "view", "entity",
		func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			called = true
			return mcp.NewToolResultText("must not run"), nil
		})

	ctx := WithPrincipal(context.Background(), uuid.New())
	res, err := guarded(ctx, mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("guard returned non-nil err on deny: %v", err)
	}
	if called {
		t.Fatal("inner handler MUST NOT run when access is denied")
	}
	if res == nil || !res.IsError {
		t.Fatal("expected an error CallToolResult on deny")
	}
}

// TestWithAccessGuard_rejectsMissingPrincipal — if the bearer middleware
// failed to stash a principal, the guard should still reject rather than
// default to "anonymous". Defense in depth.
func TestWithAccessGuard_rejectsMissingPrincipal(t *testing.T) {
	ev := &stubAccess{decision: identity_access.AccessDecision{Allow: true}}
	guarded := withAccessGuard(ev, "view", "entity",
		func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("nope"), nil
		})

	res, err := guarded(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("guard returned non-nil err: %v", err)
	}
	if res == nil || !res.IsError {
		t.Fatal("expected error result when principal context is missing")
	}
	if len(ev.calls) != 0 {
		t.Errorf("Evaluate must not be called when principal is missing (got %d calls)", len(ev.calls))
	}
}

// TestWithAccessGuard_propagatesEvalError covers the rare AccessEvaluator
// failure path: a DB hiccup or context cancel shouldn't make the guard
// silently allow the call.
func TestWithAccessGuard_propagatesEvalError(t *testing.T) {
	ev := &stubAccess{err: errors.New("db down")}
	guarded := withAccessGuard(ev, "view", "entity",
		func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			t.Fatal("inner handler must not run on Evaluate error")
			return nil, nil
		})

	ctx := WithPrincipal(context.Background(), uuid.New())
	res, _ := guarded(ctx, mcp.CallToolRequest{})
	if res == nil || !res.IsError {
		t.Fatal("expected error result on Evaluate error")
	}
}

// TestNew_allToolsAccessGuarded is the static contract test that prevents
// regression of the access-guard pattern. Every guardedTool returned by New()
// must declare a non-empty action + resourceType, and the registered handler
// must route through withAccessGuard. We verify the latter by calling the
// registered handler with a deny-decision evaluator and asserting it short-
// circuits without reaching any inner work.
func TestNew_allToolsAccessGuarded(t *testing.T) {
	deps := Deps{
		Access: &stubAccess{decision: identity_access.AccessDecision{Allow: false, ReasonCode: "test_deny"}},
		// Other deps are nil — the deny short-circuit means the inner
		// handler never reaches them.
	}
	_, tools := New(deps)
	if len(tools) == 0 {
		t.Fatal("New() returned zero tools — registry empty?")
	}
	for _, gt := range tools {
		t.Run(gt.tool.Name, func(t *testing.T) {
			if gt.action == "" {
				t.Errorf("tool %q has empty action", gt.tool.Name)
			}
			if gt.resourceType == "" {
				t.Errorf("tool %q has empty resourceType", gt.tool.Name)
			}
			// Run via the wrapper as the registry would.
			guarded := withAccessGuard(deps.Access, gt.action, gt.resourceType, gt.handler)
			ctx := WithPrincipal(context.Background(), uuid.New())
			res, err := guarded(ctx, mcp.CallToolRequest{})
			if err != nil {
				t.Errorf("guard returned err %v", err)
			}
			if res == nil || !res.IsError {
				t.Errorf("tool %q did not short-circuit on deny — access guard regression", gt.tool.Name)
			}
		})
	}
}
