package crypto

import (
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"
)

const testKeyHex = "0d71f78929e8b688442387dd10478006998c1fa490c42c02c627a3e5ec8a3bed"
const testKeyRaw = "0123456789abcdef0123456789abcdef"

func TestNewBoxKeyFormats(t *testing.T) {
	t.Run("64 hex characters", func(t *testing.T) {
		if _, err := NewBox(testKeyHex); err != nil {
			t.Fatalf("expected 64-char hex key to be accepted, got: %v", err)
		}
	})
	t.Run("32 raw characters", func(t *testing.T) {
		if _, err := NewBox(testKeyRaw); err != nil {
			t.Fatalf("expected 32-char raw key to be accepted, got: %v", err)
		}
	})
	t.Run("rejects empty", func(t *testing.T) {
		if _, err := NewBox(""); err == nil {
			t.Fatal("expected empty key to be rejected")
		}
	})
	t.Run("rejects wrong length", func(t *testing.T) {
		for _, bad := range []string{"short", strings.Repeat("a", 33), strings.Repeat("a", 63), strings.Repeat("a", 65)} {
			if _, err := NewBox(bad); err == nil {
				t.Errorf("expected %d-char key to be rejected", len(bad))
			}
		}
	})
	t.Run("rejects non-hex 64 char", func(t *testing.T) {
		if _, err := NewBox(strings.Repeat("z", 64)); err == nil {
			t.Fatal("expected non-hex 64-char key to be rejected")
		}
	})
}

// TestWireFormat pins the on-disk format. Tokens already stored in
// users.github_token_encrypted were written in this format, so changing it
// requires a data migration. This test is the guard against a second,
// incompatible decryptor being introduced alongside it.
func TestWireFormat(t *testing.T) {
	box, err := NewBox(testKeyHex)
	if err != nil {
		t.Fatal(err)
	}

	const plaintext = "ghp_abcdefgh" // exactly 12 bytes
	enc, err := box.Encrypt(plaintext)
	if err != nil {
		t.Fatal(err)
	}

	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		t.Fatalf("output must be base64: %v", err)
	}
	// 12-byte nonce + ciphertext. GCM adds a 16-byte tag, so a 12-byte
	// plaintext yields 12 + 12 + 16 = 40 bytes.
	if len(raw) != 12+12+16 {
		t.Errorf("expected 40 bytes (12 nonce + 12 ct + 16 tag), got %d", len(raw))
	}

	// Must be round-trippable.
	got, err := box.Decrypt(enc)
	if err != nil {
		t.Fatalf("round trip failed: %v", err)
	}
	if got != plaintext {
		t.Errorf("round trip mismatch: got %q", got)
	}

	// Must be non-deterministic (fresh nonce per call).
	enc2, err := box.Encrypt(plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if enc == enc2 {
		t.Error("two encryptions of the same plaintext must differ (nonce reuse)")
	}
}

func TestDecryptRejectsHexEncodedInput(t *testing.T) {
	// Regression guard: the worker previously hand-rolled a decryptor that
	// expected hex(nonce || ciphertext) with a 32-byte nonce. That format
	// never existed and silently failed on every real token while the error
	// was discarded, so private-repo clones fell back to unauthenticated.
	// The single Box must be the only decryptor, and it must speak base64.
	box, err := NewBox(testKeyHex)
	if err != nil {
		t.Fatal(err)
	}
	enc, err := box.Encrypt("gho_testtoken")
	if err != nil {
		t.Fatal(err)
	}

	// Hex-encoding a real ciphertext is what the old code tried to parse.
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := box.Decrypt(hex.EncodeToString(raw)); err == nil {
		t.Fatal("expected Box.Decrypt to reject hex-encoded input")
	}

	// And a base64 value must never be mistaken for hex input.
	if _, err := hex.DecodeString(enc); err == nil {
		t.Error("expected base64 output to be invalid hex, otherwise the old " +
			"hex-based decryptor would appear to work")
	}
}

func TestDecryptFailsAcrossKeys(t *testing.T) {
	a, _ := NewBox(testKeyHex)
	b, _ := NewBox(strings.Repeat("ab", 32))
	enc, err := a.Encrypt("gho_testtoken")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.Decrypt(enc); err == nil {
		t.Fatal("expected decryption under the wrong key to fail")
	}
}
