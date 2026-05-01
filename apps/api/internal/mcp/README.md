# mcp

Model Context Protocol endpoint mounted at `POST /mcp` (v0.5.1+). External AI clients (Claude Desktop, Cursor, IDE plugins) consume Knowledge Layer tools after authenticating through the [v0.5.0 OAuth proxy](../oauth_proxy/README.md).

Design rationale + alternatives in [ADR-0015](../../../../docs/adr/0015-oauth-proxy-and-mcp-bridge.md). Operator setup (Claude Desktop config, OIDC integration) in [docs/operations/mcp.md](../../../../docs/operations/mcp.md).

> **Hard contract: every tool MUST be wrapped in `withAccessGuard`.** The `TestNew_allToolsAccessGuarded` static-contract test iterates the registry and asserts every handler short-circuits on a deny decision. CI fails if a new tool bypasses the wrapper.

## Surface

```go
type Deps struct {
    Access    AccessEvaluator        // identity_access.AccessEvaluator (interface form for testability)
    Search    *search.Service
    Retrieval *retrieval_intelligence.Service
    Entities  *knowledge_core.EntityRepo
    ServerName, ServerVer string
}
func New(deps Deps) (*server.MCPServer, []guardedTool)

func Mount(r fiber.Router, srv *server.MCPServer, bv BearerVerifier)
type BearerVerifier interface { VerifyBearer(string) (uuid.UUID, error) }
```

## Files

- `access_guard.go` — `withAccessGuard(eval, action, resourceType, fn)` is the only legitimate way to register a tool. Reads the principal from context (set by `bearerMiddleware`); calls `AccessEvaluator.Evaluate` with the action / resource_type the tool declared at construction; deny short-circuits with an MCP-shaped error result.
- `server.go` — `New(deps)` constructs the `*mcp.MCPServer`, registers the v0.5.1 initial tool set: `kl_search`, `kl_ask_global`, `kl_get_entity`. Each goes through `newGuardedTool` then through `withAccessGuard` at registration time.
- `route.go` — Fiber `Mount` wires `POST /mcp` (and `GET /mcp` for SSE) behind `bearerMiddleware`. The middleware extracts `Authorization: Bearer X`, calls `oauth_proxy.Server.VerifyBearer`, stashes the principal UUID via `WithPrincipal(ctx, ...)` so the access-guard can read it.
- `server_test.go` — 4 unit tests for the guard (allow / deny / missing-principal / eval-error) + the static-contract `TestNew_allToolsAccessGuarded` that iterates registered tools.

## Initial tool set

| Tool | Wraps | Action | Resource type |
|---|---|---|---|
| `kl_search` | `Retrieval.SearchScoped` | `view` | `entity` |
| `kl_ask_global` | `Retrieval.AskGlobal` (privacy gateway) | `view` | `entity` |
| `kl_get_entity` | `Entities.Get` | `view` | `entity` |

All read-only in v0.5.1. Mutation tools (publish, patch) come later as the access-guard pattern proves itself in production.

## Adding a new tool

1. Construct via `newGuardedTool(name, description, action, resourceType, schema, handler)`.
2. Append to the slice in `New(deps)`.
3. The static-contract test will exercise the new tool automatically — if the handler doesn't short-circuit on deny, CI fails.

```go
buildPublishEntityTool(deps Deps) guardedTool {
    return newGuardedTool(
        "kl_publish_entity",
        "Publish a draft entity. Action publish on entity.",
        "publish", "entity",
        mcp.ToolInputSchema{ /* ... */ },
        func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            principal, _ := principalFromContext(ctx)  // guard already verified
            // ... publish logic
        },
    )
}
```

## Bearer flow

```
MCP client → POST /mcp with Authorization: Bearer <jwt>
            ↓
   bearerMiddleware
     ├─ extract Bearer token
     ├─ oauth_proxy.Server.VerifyBearer (HS256 verify, sub → uuid.UUID)
     └─ ctx = WithPrincipal(ctx, principal)
            ↓
   mcp-go StreamableHTTPServer
            ↓
   tool handler (wrapped by withAccessGuard)
     ├─ principalFromContext → uuid.UUID (or 401-ish error)
     ├─ AccessEvaluator.Evaluate(action, resource_type, principal)
     └─ allow → inner fn(ctx, req); deny → error result
```

## Audit

`mcp.tool.invoked` events log to **stderr** in v0.5.1. Promotion to `audit_events` rows is a v0.5.x follow-up.

## Why the access-guard contract is a static-grep test

A regression where someone forgets `withAccessGuard` is silent — the tool would respond to every authenticated request regardless of permission. The static-contract test (`TestNew_allToolsAccessGuarded`) iterates `New(deps)` output and asserts every handler short-circuits when given a deny-decision evaluator. This is cheap, deterministic, and catches the regression immediately.

If you split tool construction across files, ensure every new tool flows through `New(deps)`; the test only knows about tools the constructor returns.
