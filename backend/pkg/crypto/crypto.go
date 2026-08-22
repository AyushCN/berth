package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

var key []byte

func init() {
	k := os.Getenv("ENCRYPTION_KEY")
	if k == "" {
		panic("ENCRYPTION_KEY is required")
	}
	// Accept 32-byte raw key or 64-char hex string
	if len(k) == 64 {
		key = make([]byte, 32)
		for i := 0; i < 32; i++ {
			b, _ := fmt.Sscanf(k[i*2:i*2+2], "%02x", &key[i])
			_ = b
		}
	} else if len(k) == 32 {
		key = []byte(k)
	} else {
		panic("ENCRYPTION_KEY must be 32 bytes raw or 64 chars hex")
	}
}

// Encrypt encrypts plaintext with AES-256-GCM and returns base64(ciphertext||nonce).
func Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64(ciphertext||nonce) and returns plaintext.
func Decrypt(ciphertextB64 string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	if len(ciphertext) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}
