package privacy

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// VaultCodec encrypts placeholder cleartext for ai_placeholder_mappings.
type VaultCodec struct {
	key          []byte
	devPlaintext bool
}

// NewVaultCodecFromEnv loads AES-256 key from AI_PRIVACY_VAULT_KEY (base64 or raw 32 bytes).
// If AI_PRIVACY_DEV_PLAINTEXT_STORE=1, stores UTF-8 "plaintext" in ciphertext column (local dev only).
func NewVaultCodecFromEnv() (*VaultCodec, error) {
	if strings.TrimSpace(os.Getenv("AI_PRIVACY_DEV_PLAINTEXT_STORE")) == "1" {
		return &VaultCodec{devPlaintext: true}, nil
	}
	raw := strings.TrimSpace(os.Getenv("AI_PRIVACY_VAULT_KEY"))
	if raw == "" {
		return nil, errors.New("privacy vault: AI_PRIVACY_VAULT_KEY or AI_PRIVACY_DEV_PLAINTEXT_STORE=1 required")
	}
	var key []byte
	if b, err := base64.StdEncoding.DecodeString(raw); err == nil && len(b) == 32 {
		key = b
	} else if len(raw) == 32 {
		key = []byte(raw)
	} else {
		return nil, fmt.Errorf("privacy vault: AI_PRIVACY_VAULT_KEY must be 32 raw bytes or base64 of 32 bytes")
	}
	return &VaultCodec{key: key}, nil
}

// Encrypt returns nonce+ciphertext for storage (nonce 12 bytes prepended to ciphertext in DB columns separately).
func (c *VaultCodec) Encrypt(plaintext string) (nonce, ciphertext []byte, err error) {
	if c == nil {
		return nil, nil, errors.New("privacy vault: nil codec")
	}
	if c.devPlaintext {
		return []byte("DEVPLAIN"), []byte(plaintext), nil
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	return nonce, gcm.Seal(nil, nonce, []byte(plaintext), nil), nil
}

// Decrypt reverses Encrypt.
func (c *VaultCodec) Decrypt(nonce, ciphertext []byte) (string, error) {
	if c == nil {
		return "", errors.New("privacy vault: nil codec")
	}
	if c.devPlaintext {
		if string(nonce) == "DEVPLAIN" {
			return string(ciphertext), nil
		}
		return "", errors.New("privacy vault: invalid dev plaintext row")
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(nonce) != gcm.NonceSize() {
		return "", errors.New("privacy vault: bad nonce size")
	}
	out, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
