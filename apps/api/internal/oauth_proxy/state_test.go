package oauth_proxy

import (
	"strings"
	"testing"
	"time"
)

// TestSignVerify_roundTrip is the happy path: a freshly-signed payload
// verifies cleanly with the same key, and the decoded value matches the
// original byte-for-byte.
func TestSignVerify_roundTrip(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef") // 32 bytes
	in := Payload{
		ClientID:            "mcp-client-1",
		RedirectURI:         "http://localhost:8081/callback",
		CodeChallenge:       "abc",
		CodeChallengeMethod: "S256",
		Scope:               "openid profile",
		Nonce:               "nonce-1",
	}
	tok, err := Sign(in, key)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	out, err := Verify(tok, key)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if out.ClientID != in.ClientID || out.CodeChallenge != in.CodeChallenge ||
		out.RedirectURI != in.RedirectURI || out.Scope != in.Scope || out.Nonce != in.Nonce {
		t.Errorf("round-trip mismatch:\n  in:  %+v\n  out: %+v", in, *out)
	}
}

// TestVerify_tamperedBody changes one byte in the body and asserts Verify
// rejects it. This is the primary attack we defend against — an attacker
// editing the redirect_uri or scope after the IDP authenticated the user.
func TestVerify_tamperedBody(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	tok, _ := Sign(Payload{ClientID: "c1", RedirectURI: "http://localhost"}, key)
	parts := strings.SplitN(tok, ".", 2)
	if len(parts) != 2 {
		t.Fatalf("unexpected token shape: %q", tok)
	}
	// Flip one body char so the HMAC no longer matches.
	mutated := parts[0][:len(parts[0])-1] + "Z" + "." + parts[1]
	if _, err := Verify(mutated, key); err == nil {
		t.Fatal("Verify accepted tampered body")
	}
}

// TestVerify_wrongKey checks the obvious rotation case: a token issued under
// one key cannot be verified after a key rotation. Operators rely on this to
// invalidate in-flight authorizations during incident response.
func TestVerify_wrongKey(t *testing.T) {
	keyA := []byte("0123456789abcdef0123456789abcdef")
	keyB := []byte("ffffffffffffffffffffffffffffffff")
	tok, _ := Sign(Payload{ClientID: "c1", RedirectURI: "http://localhost"}, keyA)
	if _, err := Verify(tok, keyB); err == nil {
		t.Fatal("Verify accepted token signed under different key")
	}
}

// TestVerify_expired ensures stateMaxAge is honored — an attacker who steals
// a state value can't replay it indefinitely.
func TestVerify_expired(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	old := time.Now().Add(-stateMaxAge - time.Minute).Unix()
	in := Payload{ClientID: "c1", RedirectURI: "http://localhost", IssuedAt: old}
	tok, _ := Sign(in, key)
	if _, err := Verify(tok, key); err == nil {
		t.Fatal("Verify accepted expired state")
	}
}

// TestSign_rejectsShortKey enforces the 32-byte minimum. Operators who set
// OAUTH_SECRET_KEY to a guessable string get a clear error at startup.
func TestSign_rejectsShortKey(t *testing.T) {
	if _, err := Sign(Payload{}, []byte("too short")); err == nil {
		t.Fatal("Sign accepted a key < 32 bytes")
	}
	if _, err := Verify("a.b", []byte("too short")); err == nil {
		t.Fatal("Verify accepted a key < 32 bytes")
	}
}

// TestVerify_malformed covers non-token inputs that should be rejected
// without panic.
func TestVerify_malformed(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	for _, in := range []string{"", ".", "no-dot", ".starting-dot", "ending-dot."} {
		if _, err := Verify(in, key); err == nil {
			t.Errorf("Verify accepted malformed %q", in)
		}
	}
}
