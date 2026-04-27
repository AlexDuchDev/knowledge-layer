package privacy

import (
	"crypto/rand"
	"testing"
)

func TestVaultCodecEncryptRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	c := &VaultCodec{key: key}
	nonce, ct, err := c.Encrypt("super-secret-value")
	if err != nil {
		t.Fatal(err)
	}
	plain, err := c.Decrypt(nonce, ct)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "super-secret-value" {
		t.Fatalf("got %q", plain)
	}
}

func TestVaultCodecDevPlaintext(t *testing.T) {
	c := &VaultCodec{devPlaintext: true}
	n, ct, err := c.Encrypt("hello")
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.Decrypt(n, ct)
	if err != nil || s != "hello" {
		t.Fatalf("%q %v", s, err)
	}
}
