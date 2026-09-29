// Package crypto provides symmetric encryption for secrets at rest
// (currently GitHub OAuth tokens stored in users.github_token_encrypted).
//
// The key is supplied explicitly by the caller rather than read from the
// environment at package init time. Reading the environment in init() runs
// before main() and therefore before config.Load() and godotenv, which made
// both binaries panic on startup whenever ENCRYPTION_KEY was not present in
// the real process environment.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

// Box encrypts and decrypts secrets with AES-256-GCM.
//
// Wire format (do not change without a data migration):
//
//	base64.StdEncoding( nonce[12] || ciphertext )
//
// The nonce is gcm.NonceSize() and is prepended to the sealed output by
// gcm.Seal, then the whole buffer is base64 encoded.
type Box struct {
	aead cipher.AEAD
}

// NewBox derives a 32-byte key from secret and returns a ready Box.
//
// Two key formats are accepted for backwards compatibility with keys that
// were generated before this was centralised:
//   - 64 hex characters  -> decoded to 32 bytes
//   - 32 raw characters  -> used as the 32 key bytes directly
//
// Anything else is an error. Callers should fail fast on a bad key at
// startup rather than silently degrading to an empty key.
func NewBox(secret string) (*Box, error) {
	if secret == "" {
		return nil, fmt.Errorf("encryption key is required")
	}

	var key []byte
	switch len(secret) {
	case 64:
		decoded, err := hex.DecodeString(secret)
		if err != nil {
			return nil, fmt.Errorf("encryption key is not valid hex: %w", err)
		}
		key = decoded
	case 32:
		key = []byte(secret)
	default:
		return nil, fmt.Errorf("encryption key must be 32 raw bytes or 64 hex characters, got %d characters", len(secret))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	return &Box{aead: aead}, nil
}

// Encrypt seals plaintext and returns base64(nonce || ciphertext).
func (b *Box) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	// Seal appends the ciphertext to nonce, giving nonce || ciphertext.
	sealed := b.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt opens a value produced by Encrypt.
func (b *Box) Decrypt(encoded string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}
	ns := b.aead.NonceSize()
	if len(raw) < ns {
		return "", fmt.Errorf("ciphertext too short: %d bytes, need at least %d", len(raw), ns)
	}
	nonce, ciphertext := raw[:ns], raw[ns:]
	plaintext, err := b.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}
	return string(plaintext), nil
}
