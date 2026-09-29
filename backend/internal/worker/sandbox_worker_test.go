package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/AyushCN/berth/pkg/crypto"
)

const testKey = "0d71f78929e8b688442387dd10478006998c1fa490c42c02c627a3e5ec8a3bed"

type stubUserRepo struct {
	user *domain.User
	err  error
}

func (s *stubUserRepo) Create(context.Context, *domain.User) error { return nil }
func (s *stubUserRepo) GetByID(context.Context, uuid.UUID) (*domain.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.user, nil
}
func (s *stubUserRepo) GetByGithubID(context.Context, string) (*domain.User, error) { return nil, nil }
func (s *stubUserRepo) Update(context.Context, *domain.User) error                    { return nil }

// resolveGitToken is the fix for the hand-rolled decryptor that expected
// hex(nonce || ciphertext) with a 32-byte nonce. That format was never
// produced by crypto.Encrypt, so every private-repo clone silently fell back
// to unauthenticated while still logging "using OAuth token".
func TestResolveGitTokenDecryptsBoxOutput(t *testing.T) {
	box, err := crypto.NewBox(testKey)
	if err != nil {
		t.Fatal(err)
	}

	// Encrypt exactly the way usecase/auth.go does on GitHub login.
	encrypted, err := box.Encrypt("gho_realtoken")
	if err != nil {
		t.Fatal(err)
	}

	ownerID := uuid.New()
	w := &SandboxWorker{
		userRepo: &stubUserRepo{user: &domain.User{
			ID:                   ownerID,
			GithubTokenEncrypted: encrypted,
		}},
		tokenBox: box,
	}

	got, err := w.resolveGitToken(context.Background(), ownerID)
	if err != nil {
		t.Fatalf("failed to decrypt a token produced by crypto.Encrypt: %v", err)
	}
	if got != "gho_realtoken" {
		t.Fatalf("got %q, want %q", got, "gho_realtoken")
	}
}

func TestResolveGitTokenNoTokenIsNotAnError(t *testing.T) {
	box, _ := crypto.NewBox(testKey)
	ownerID := uuid.New()

	// Public repo with no stored token must clone unauthenticated, not fail.
	w := &SandboxWorker{userRepo: &stubUserRepo{user: &domain.User{ID: ownerID}}, tokenBox: box}
	got, err := w.resolveGitToken(context.Background(), ownerID)
	if err != nil || got != "" {
		t.Fatalf("expected empty token and no error, got %q / %v", got, err)
	}
}

func TestResolveGitTokenNilSafe(t *testing.T) {
	// Must not panic when the worker was built without a box or owner.
	w := &SandboxWorker{}
	if got, err := w.resolveGitToken(context.Background(), uuid.Nil); err != nil || got != "" {
		t.Fatalf("expected empty token and no error, got %q / %v", got, err)
	}
	if got, err := w.resolveGitToken(context.Background(), uuid.New()); err != nil || got != "" {
		t.Fatalf("expected empty token and no error, got %q / %v", got, err)
	}
}

func TestResolveGitTokenWrongKeyIsAnError(t *testing.T) {
	good, _ := crypto.NewBox(testKey)
	other, _ := crypto.NewBox("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	encrypted, _ := good.Encrypt("gho_realtoken")

	ownerID := uuid.New()
	w := &SandboxWorker{
		userRepo: &stubUserRepo{user: &domain.User{ID: ownerID, GithubTokenEncrypted: encrypted}},
		tokenBox: other,
	}
	if _, err := w.resolveGitToken(context.Background(), ownerID); err == nil {
		t.Fatal("expected decryption under a mismatched key to error")
	}
}

func TestResolveGitTokenRepoErrorPropagates(t *testing.T) {
	box, _ := crypto.NewBox(testKey)
	ownerID := uuid.New()
	w := &SandboxWorker{
		userRepo: &stubUserRepo{err: errors.New("db down")},
		tokenBox: box,
	}
	if _, err := w.resolveGitToken(context.Background(), ownerID); err == nil {
		t.Fatal("expected repository error to propagate")
	}
}
