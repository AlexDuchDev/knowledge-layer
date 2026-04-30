package mcp

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/knowledgelayer/api/internal/knowledge_core"
	"github.com/knowledgelayer/api/internal/qa"
	"github.com/knowledgelayer/api/internal/retrieval_intelligence"
	"github.com/knowledgelayer/api/internal/search"
)

// Deps wires the MCP package to the existing services. Three initial tools
// (kl_search, kl_ask_global, kl_get_entity) are surfaced; each is access-
// guarded against AccessEvaluator. v0.5.x point releases add more tools as
// the access-guard pattern proves itself.
type Deps struct {
	Access     AccessEvaluator
	Search     *search.Service
	Retrieval  *retrieval_intelligence.Service
	Entities   *knowledge_core.EntityRepo
	ServerName string // Default: "knowledge-layer-mcp"
	ServerVer  string // Build version string
}

// New builds a fully wired MCP server with all initial tools registered
// through withAccessGuard. The returned server can be transported via
// streamable HTTP (Mount), stdio, or any transport mcp-go supports.
//
// Important: tool registration order is meaningful for tests — server_test.go
// iterates the registered tools and asserts every one is access-guarded.
func New(deps Deps) (*server.MCPServer, []guardedTool) {
	name := deps.ServerName
	if name == "" {
		name = "knowledge-layer-mcp"
	}
	ver := deps.ServerVer
	if ver == "" {
		ver = "dev"
	}
	srv := server.NewMCPServer(name, ver, server.WithToolCapabilities(false))

	// Build the registry, then attach. Two-step so the test can run the
	// "every tool wraps Evaluate" assertion against the registry without
	// inventing a parallel listing.
	tools := []guardedTool{
		buildSearchTool(deps),
		buildAskGlobalTool(deps),
		buildGetEntityTool(deps),
	}
	for _, t := range tools {
		guarded := withAccessGuard(deps.Access, t.action, t.resourceType, t.handler)
		srv.AddTool(t.tool, guarded)
	}
	return srv, tools
}

// buildSearchTool wraps the existing /search keyword path. Action="view"
// resourceType="entity" — AccessEvaluator decides whether the principal
// can run a search at all; per-result domain scoping happens inside the
// search service via permissions.Resolver.
func buildSearchTool(deps Deps) guardedTool {
	schema := mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]any{
			"q":          map[string]any{"type": "string", "description": "Keyword query string"},
			"type":       map[string]any{"type": "string", "description": "Restrict to one entity type (Decision, Insight, ReferenceDocument, ...)"},
			"domain_id":  map[string]any{"type": "string", "description": "Restrict to one domain UUID"},
			"limit":      map[string]any{"type": "integer", "description": "Max results (default 20, hard cap 100)"},
		},
		Required: []string{"q"},
	}
	return newGuardedTool(
		"kl_search",
		"Keyword search across published Knowledge Layer entities the caller can view. Returns entity summaries with citations.",
		"view", "entity", schema,
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			principal, err := principalFromContext(ctx)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			args := req.GetArguments()
			filters := map[string]string{}
			if v, ok := args["q"].(string); ok {
				filters["q"] = v
			}
			if v, ok := args["type"].(string); ok && v != "" {
				filters["type"] = v
			}
			if v, ok := args["domain_id"].(string); ok && v != "" {
				filters["domain_id"] = v
			}
			hits, err := deps.Retrieval.SearchScoped(ctx, principal, filters)
			if err != nil {
				return mcp.NewToolResultError("search failed: " + err.Error()), nil
			}
			limit := 20
			if v, ok := args["limit"].(float64); ok && int(v) > 0 && int(v) <= 100 {
				limit = int(v)
			}
			if len(hits) > limit {
				hits = hits[:limit]
			}
			return mcp.NewToolResultStructuredOnly(map[string]any{"hits": hits, "count": len(hits)}), nil
		},
	)
}

// buildAskGlobalTool exposes /ask global Q&A. Action="view" resourceType="entity"
// for the same reason — AccessEvaluator confirms the user can access the
// memory at all; per-document permission filtering already happens inside
// retrieval_intelligence and qa.
func buildAskGlobalTool(deps Deps) guardedTool {
	schema := mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]any{
			"question":          map[string]any{"type": "string", "description": "Natural-language question"},
			"include_related":   map[string]any{"type": "boolean", "description": "Include related entities in evidence"},
		},
		Required: []string{"question"},
	}
	return newGuardedTool(
		"kl_ask_global",
		"Synthesize an answer from organizational memory. Goes through the same privacy gateway and access checks as the /ask REST endpoint. Returns the answer plus citations.",
		"view", "entity", schema,
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			principal, err := principalFromContext(ctx)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			args := req.GetArguments()
			question, _ := args["question"].(string)
			includeRelated, _ := args["include_related"].(bool)

			searchFilters := map[string]string{"q": question}

			// retrieval_intelligence.AskGlobal builds the canView callback
			// internally via the centralized 9-step AccessEvaluator (Phase 1
			// alignment). The MCP tool just supplies the question + filters.
			ans, _, err := deps.Retrieval.AskGlobal(ctx, principal, qa.AskEntityInput{
				Question:       question,
				IncludeRelated: includeRelated,
			}, searchFilters, "hybrid")
			if err != nil {
				return mcp.NewToolResultError("ask failed: " + err.Error()), nil
			}
			return mcp.NewToolResultStructuredOnly(ans), nil
		},
	)
}

// buildGetEntityTool wraps GET /entities/:id. Action="view" resourceType="entity";
// EntityID is also passed in so the evaluator can do per-entity access (the
// 9-step evaluator already encodes resource-id-aware decisions).
func buildGetEntityTool(deps Deps) guardedTool {
	schema := mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]any{
			"entity_id": map[string]any{"type": "string", "description": "Entity UUID"},
		},
		Required: []string{"entity_id"},
	}
	return newGuardedTool(
		"kl_get_entity",
		"Fetch a single Knowledge Layer entity by id. Returns title, body, lifecycle state, type, owner. Permission-checked via AccessEvaluator.",
		"view", "entity", schema,
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			idStr, _ := args["entity_id"].(string)
			id, err := parseUUID(idStr)
			if err != nil {
				return mcp.NewToolResultError("invalid entity_id: " + err.Error()), nil
			}
			ent, err := deps.Entities.Get(ctx, id)
			if err != nil {
				return mcp.NewToolResultError("get_entity failed: " + err.Error()), nil
			}
			return mcp.NewToolResultStructuredOnly(map[string]any{
				"id":              ent.ID.String(),
				"type":            ent.Type,
				"title":           ent.Title,
				"summary":         ent.Summary,
				"body":            ent.Body,
				"lifecycle_state": ent.LifecycleState,
				"truth_mode":      ent.TruthMode,
				"domain_id":       ent.DomainID.String(),
			}), nil
		},
	)
}

// parseUUID validates and parses a string ID supplied by an MCP client.
func parseUUID(s string) (uuid.UUID, error) {
	if s == "" {
		return uuid.Nil, fmt.Errorf("empty")
	}
	return uuid.Parse(s)
}
