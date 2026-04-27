// Package prompts is the central registry for LLM prompt templates used by
// Knowledge Layer (Phase 4.1.1).
//
// Why a registry: ad-hoc prompts inside synthesis code are invisible to
// audit, hard to version, and silently drift when a behaviour change is
// shipped without a deliberate template bump. The registry gives every prompt:
//
//   - A stable, versioned ID (`ask_global_qa.v1`) that lands in audit/trace.
//   - A single .json source per (id, version) under templates/.
//   - Named-parameter substitution so call sites cannot accidentally splice
//     unsanitized data into the system prompt.
//   - Compile-time embedding (`go:embed`) — no DB lookup, no migration drift,
//     no admin UI surface to maintain. Templates are part of the binary.
//
// When to add a template:
//   - Any new LLM-using flow.
//   - Any time a prompt change would observably alter outputs (treat it as a
//     contract change: bump .v2, ship the new file, retire the old per the
//     deprecation cycle).
//
// When NOT to use this:
//   - Tiny one-shot system messages that are part of an integration test —
//     hardcode in the test.
//   - Cross-cutting style hints that apply to every flow (those belong in
//     a shared rules block referenced from multiple templates).
package prompts

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

//go:embed templates/*.json
var fs embed.FS

// Prompt is one versioned template loaded from templates/<id>.json.
//
// ID is the canonical reference used in audit trails (e.g. "ask_global_qa.v1").
// Bump the suffix when SystemPrompt or UserTemplate change in a way callers
// would observe; never repurpose an existing version.
type Prompt struct {
	ID           string `json:"id"`
	Description  string `json:"description"`
	SystemPrompt string `json:"system_prompt"`
	UserTemplate string `json:"user_template"`
}

// Render produces the user-message text by substituting `{{name}}` placeholders
// with the supplied params. Unknown placeholders are left as literal `{{name}}`
// so a typo never silently drops content; the call site can grep the rendered
// string in tests to assert no leftover placeholders.
func (p Prompt) Render(params map[string]string) string {
	if p.UserTemplate == "" {
		return ""
	}
	out := p.UserTemplate
	for k, v := range params {
		out = strings.ReplaceAll(out, "{{"+k+"}}", v)
	}
	return out
}

var (
	once     sync.Once
	cache    = map[string]Prompt{}
	cacheErr error
)

// Get returns a registered prompt by ID. Templates are loaded once at first
// access and held in memory for the process lifetime — they are immutable
// after build, so no invalidation is needed.
func Get(id string) (Prompt, error) {
	once.Do(load)
	if cacheErr != nil {
		return Prompt{}, cacheErr
	}
	p, ok := cache[id]
	if !ok {
		return Prompt{}, fmt.Errorf("prompts: unknown id %q (registered: %v)", id, IDs())
	}
	return p, nil
}

// MustGet panics on missing prompt — only use at process startup or in tests
// where a missing template is unrecoverable.
func MustGet(id string) Prompt {
	p, err := Get(id)
	if err != nil {
		panic(err)
	}
	return p
}

// IDs lists every registered prompt ID; helpful for "did you mean…" hints
// and for /metrics-style introspection.
func IDs() []string {
	once.Do(load)
	out := make([]string, 0, len(cache))
	for id := range cache {
		out = append(out, id)
	}
	return out
}

// load reads every templates/*.json into the cache. Called exactly once via
// sync.Once — concurrent first-callers wait for the same load.
func load() {
	entries, err := fs.ReadDir("templates")
	if err != nil {
		cacheErr = fmt.Errorf("prompts: read templates dir: %w", err)
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := fs.ReadFile("templates/" + e.Name())
		if err != nil {
			cacheErr = fmt.Errorf("prompts: read %s: %w", e.Name(), err)
			return
		}
		var p Prompt
		if err := json.Unmarshal(b, &p); err != nil {
			cacheErr = fmt.Errorf("prompts: parse %s: %w", e.Name(), err)
			return
		}
		if p.ID == "" || p.SystemPrompt == "" {
			cacheErr = fmt.Errorf("prompts: %s missing required fields (id, system_prompt)", e.Name())
			return
		}
		if existing, dup := cache[p.ID]; dup {
			cacheErr = fmt.Errorf("prompts: duplicate id %q (also in %s)", p.ID, existing.ID)
			return
		}
		cache[p.ID] = p
	}
}
