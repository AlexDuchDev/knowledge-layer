// Package mcp implements the Model Context Protocol endpoint that lets
// external AI clients (Claude Desktop, Cursor, IDE plugins) consume Knowledge
// Layer tools after authenticating through the v0.5.0 OAuth proxy.
//
// Hard contract: every tool registered with the MCP server MUST be wrapped in
// withAccessGuard. The wrapper calls AccessEvaluator.Evaluate(...) with an
// action / resource_type derived from the tool's metadata; a deny halts the
// call and returns an MCP-shaped error to the client. There is no shortcut,
// admin bypass, or "trust the bearer" path. ADR-0015 codifies this.
//
// To add a new tool, declare it via newGuardedTool(name, action, resourceType,
// schema, fn). The static-grep test (server_test.go) asserts every tool in the
// registry was created through that constructor.
package mcp

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/knowledgelayer/api/internal/identity_access"
)

// principalContextKey is the context key the bearer middleware uses to stash
// the resolved KL principal UUID. The MCP server reads it on every tool call
// to build EvaluateInput.
type principalContextKey struct{}

// WithPrincipal stores the resolved principal in a context. Used by the
// bearer middleware before forwarding the request to the MCP handler.
func WithPrincipal(ctx context.Context, principal uuid.UUID) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

// principalFromContext extracts the principal placed by WithPrincipal.
// Returns uuid.Nil and an error if absent — the access guard rejects the
// call rather than defaulting to "anonymous".
func principalFromContext(ctx context.Context) (uuid.UUID, error) {
	v := ctx.Value(principalContextKey{})
	if v == nil {
		return uuid.Nil, errors.New("mcp: no principal in context (bearer middleware did not run?)")
	}
	id, ok := v.(uuid.UUID)
	if !ok || id == uuid.Nil {
		return uuid.Nil, errors.New("mcp: principal context value is not a uuid")
	}
	return id, nil
}

// AccessEvaluator is the subset of identity_access.AccessEvaluator the guard
// uses. Interface form so tests can pass a stub without spinning up the real
// 9-step evaluator.
type AccessEvaluator interface {
	Evaluate(ctx context.Context, in identity_access.EvaluateInput) (*identity_access.AccessDecision, error)
}

// guardedTool bundles an MCP tool descriptor with the AccessEvaluator inputs
// the guard will pass per call. It is the only way a tool enters the
// registry; the static-grep test asserts no AddTool call bypasses it.
type guardedTool struct {
	tool         mcp.Tool
	action       string // AccessEvaluator action (e.g. "view", "publish")
	resourceType string // AccessEvaluator resource_type (e.g. "entity")
	handler      server.ToolHandlerFunc
}

// newGuardedTool is the canonical constructor. Every MCP tool MUST go
// through it. Returns a guardedTool the registry helper attaches to the
// server with the access wrapper.
func newGuardedTool(name, description, action, resourceType string, schema mcp.ToolInputSchema, fn server.ToolHandlerFunc) guardedTool {
	t := mcp.NewTool(name, mcp.WithDescription(description))
	t.InputSchema = schema
	return guardedTool{
		tool:         t,
		action:       action,
		resourceType: resourceType,
		handler:      fn,
	}
}

// withAccessGuard wraps a ToolHandlerFunc so AccessEvaluator runs before the
// inner handler sees the call. Denials produce an MCP-shaped error result;
// permit lets the inner handler run with the same context (so the handler
// can re-read the principal from context for its DB queries).
func withAccessGuard(eval AccessEvaluator, action, resourceType string, fn server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		principal, err := principalFromContext(ctx)
		if err != nil {
			return mcp.NewToolResultError("unauthorized: " + err.Error()), nil
		}
		decision, err := eval.Evaluate(ctx, identity_access.EvaluateInput{
			PrincipalID:  principal,
			Action:       action,
			ResourceType: resourceType,
		})
		if err != nil {
			return mcp.NewToolResultError("access evaluation failed: " + err.Error()), nil
		}
		if decision == nil || !decision.Allow {
			reason := "permission denied"
			if decision != nil && decision.ReasonCode != "" {
				reason = fmt.Sprintf("permission denied (%s)", decision.ReasonCode)
			}
			return mcp.NewToolResultError(reason), nil
		}
		return fn(ctx, req)
	}
}
