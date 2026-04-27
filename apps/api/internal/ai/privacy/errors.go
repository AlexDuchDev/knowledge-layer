package privacy

import "errors"

// ErrDisallowAI means detected content matches a policy rule with action disallow_ai.
var ErrDisallowAI = errors.New("privacy: llm invocation disallowed by sensitive data policy")
