package usecase

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

// The file endpoints performed no authorization at all before this change, so
// any authenticated user could read, overwrite or delete any environment's
// files by UUID. These tests pin the access rules.

type fakeEnvRepo struct{ env *domain.Environment }

func (f *fakeEnvRepo) Create(context.Context, *domain.Environment) error { return nil }
func (f *fakeEnvRepo) GetByID(context.Context, uuid.UUID) (*domain.Environment, error) {
	return f.env, nil
}
func (f *fakeEnvRepo) GetByWorkspace(context.Context, uuid.UUID) ([]*domain.Environment, error) {
	return nil, nil
}
func (f *fakeEnvRepo) GetActiveByWorkspace(context.Context, uuid.UUID) ([]*domain.Environment, error) {
	return nil, nil
}
func (f *fakeEnvRepo) Update(context.Context, *domain.Environment) error { return nil }
func (f *fakeEnvRepo) UpdateState(context.Context, uuid.UUID, domain.EnvironmentState) error {
	return nil
}
func (f *fakeEnvRepo) UpdateContainerID(context.Context, uuid.UUID, string) error { return nil }
func (f *fakeEnvRepo) UpdateImageID(context.Context, uuid.UUID, uuid.UUID) error  { return nil }
func (f *fakeEnvRepo) UpdateActivity(context.Context, uuid.UUID, int) error       { return nil }
func (f *fakeEnvRepo) Delete(context.Context, uuid.UUID) error                    { return nil }
func (f *fakeEnvRepo) ListByState(context.Context, domain.EnvironmentState) ([]*domain.Environment, error) {
	return nil, nil
}
func (f *fakeEnvRepo) ListSuspended(context.Context, time.Time) ([]*domain.Environment, error) {
	return nil, nil
}
func (f *fakeEnvRepo) ListIdleRunning(context.Context, time.Time) ([]*domain.Environment, error) {
	return nil, nil
}
func (f *fakeEnvRepo) CountByStateAndRuntimeProfile(context.Context, string, uuid.UUID) (int64, error) {
	return 0, nil
}

type fakeWSRepo struct{ ws *domain.Workspace }

func (f *fakeWSRepo) Create(context.Context, *domain.Workspace) error { return nil }
func (f *fakeWSRepo) GetByID(context.Context, uuid.UUID) (*domain.Workspace, error) {
	return f.ws, nil
}
func (f *fakeWSRepo) GetByProject(context.Context, uuid.UUID) ([]*domain.Workspace, error) {
	return nil, nil
}
func (f *fakeWSRepo) GetForks(context.Context, uuid.UUID) ([]*domain.Workspace, error) {
	return nil, nil
}
func (f *fakeWSRepo) GetCanonical(context.Context, uuid.UUID) (*domain.Workspace, error) {
	return nil, nil
}
func (f *fakeWSRepo) Update(context.Context, *domain.Workspace) error { return nil }
func (f *fakeWSRepo) Delete(context.Context, uuid.UUID) error         { return nil }
func (f *fakeWSRepo) GetUserWorkspaces(context.Context, uuid.UUID) ([]*domain.Workspace, error) {
	return nil, nil
}

type fakeProjRepo struct {
	collab *domain.ProjectCollaborator
	found  bool
}

func (f *fakeProjRepo) Create(context.Context, *domain.Project) error { return nil }
func (f *fakeProjRepo) GetByID(context.Context, uuid.UUID) (*domain.Project, error) {
	return &domain.Project{ID: uuid.New()}, nil
}
func (f *fakeProjRepo) ListForOrg(context.Context, uuid.UUID) ([]*domain.Project, error) {
	return nil, nil
}
func (f *fakeProjRepo) ListForUser(context.Context, uuid.UUID) ([]*domain.Project, error) {
	return nil, nil
}
func (f *fakeProjRepo) AddCollaborator(context.Context, *domain.ProjectCollaborator) error {
	return nil
}
func (f *fakeProjRepo) GetCollaborator(context.Context, uuid.UUID, uuid.UUID) (*domain.ProjectCollaborator, error) {
	if !f.found {
		return nil, context.Canceled
	}
	return f.collab, nil
}
func (f *fakeProjRepo) ListCollaborators(context.Context, uuid.UUID) ([]*domain.ProjectCollaborator, error) {
	return nil, nil
}
func (f *fakeProjRepo) UpdateRole(context.Context, uuid.UUID, uuid.UUID, domain.ProjectRole) error {
	return nil
}
func (f *fakeProjRepo) RemoveCollaborator(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (f *fakeProjRepo) ListEnvironments(context.Context, uuid.UUID) ([]*domain.Environment, error) {
	return nil, nil
}

func fixture(owner uuid.UUID, projectID uuid.UUID, role domain.ProjectRole, collabFound bool) (domain.EnvironmentRepository, domain.WorkspaceRepository, domain.ProjectRepository) {
	env := &domain.Environment{ID: uuid.New(), WorkspaceID: uuid.New(), State: domain.EnvironmentStateRunning}
	ws := &domain.Workspace{ID: env.WorkspaceID, OwnerID: owner, ProjectID: projectID}
	return &fakeEnvRepo{env: env}, &fakeWSRepo{ws: ws}, &fakeProjRepo{
		collab: &domain.ProjectCollaborator{Role: role},
		found:  collabFound,
	}
}

func TestResolveAccess(t *testing.T) {
	owner := uuid.New()
	stranger := uuid.New()
	project := uuid.New()

	t.Run("owner gets write", func(t *testing.T) {
		e, w, p := fixture(owner, project, domain.ProjectRoleViewer, true)
		got, err := resolveAccess(context.Background(), e, w, p, uuid.New(), owner)
		if err != nil || got != accessWrite {
			t.Fatalf("got %v / %v, want accessWrite", got, err)
		}
	})

	t.Run("editor collaborator gets write", func(t *testing.T) {
		e, w, p := fixture(owner, project, domain.ProjectRoleCollaborator, true)
		got, _ := resolveAccess(context.Background(), e, w, p, uuid.New(), stranger)
		if got != accessWrite {
			t.Fatalf("got %v, want accessWrite", got)
		}
	})

	t.Run("viewer collaborator gets read only", func(t *testing.T) {
		e, w, p := fixture(owner, project, domain.ProjectRoleViewer, true)
		got, _ := resolveAccess(context.Background(), e, w, p, uuid.New(), stranger)
		if got != accessRead {
			t.Fatalf("got %v, want accessRead", got)
		}
	})

	t.Run("non-collaborator gets nothing", func(t *testing.T) {
		e, w, p := fixture(owner, project, domain.ProjectRoleViewer, false)
		got, _ := resolveAccess(context.Background(), e, w, p, uuid.New(), stranger)
		if got != accessNone {
			t.Fatalf("got %v, want accessNone", got)
		}
	})

	t.Run("stranger on a workspace with no project gets nothing", func(t *testing.T) {
		e, w, p := fixture(owner, uuid.Nil, domain.ProjectRoleViewer, true)
		got, _ := resolveAccess(context.Background(), e, w, p, uuid.New(), stranger)
		if got != accessNone {
			t.Fatalf("got %v, want accessNone", got)
		}
	})
}

// A 403 would confirm the environment exists, letting an attacker enumerate
// ids, so refusals must read as "not found".
func TestRequireAccessRefusesAsNotFound(t *testing.T) {
	owner := uuid.New()
	stranger := uuid.New()
	e, w, p := fixture(owner, uuid.New(), domain.ProjectRoleViewer, false)

	err := requireAccess(context.Background(), e, w, p, uuid.New(), stranger, accessRead, "read")
	if err == nil {
		t.Fatal("expected a refusal")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected a not-found error, got %q", err.Error())
	}
}

func TestFileUsecaseRefusesNonCollaborator(t *testing.T) {
	owner := uuid.New()
	stranger := uuid.New()
	e, w, p := fixture(owner, uuid.New(), domain.ProjectRoleViewer, false)

	// Point the workspace root at a temp dir so a failure can only be authz.
	fu := NewFileUsecase(t.TempDir(), e, w, p, nil)

	if _, err := fu.ListFiles(context.Background(), uuid.New(), stranger, "."); err == nil {
		t.Error("ListFiles must refuse a non-collaborator")
	}
	if _, err := fu.GetFileContent(context.Background(), uuid.New(), stranger, "README.md"); err == nil {
		t.Error("GetFileContent must refuse a non-collaborator")
	}
	if _, err := fu.UpdateFileContent(context.Background(), uuid.New(), stranger, "README.md", []byte("x")); err == nil {
		t.Error("UpdateFileContent must refuse a non-collaborator")
	}
	if err := fu.CreateFile(context.Background(), uuid.New(), stranger, "new.txt", false); err == nil {
		t.Error("CreateFile must refuse a non-collaborator")
	}
	if err := fu.DeleteFile(context.Background(), uuid.New(), stranger, "new.txt"); err == nil {
		t.Error("DeleteFile must refuse a non-collaborator")
	}
}

func TestFileUsecaseViewerCannotWrite(t *testing.T) {
	owner := uuid.New()
	viewer := uuid.New()
	e, w, p := fixture(owner, uuid.New(), domain.ProjectRoleViewer, true)
	fu := NewFileUsecase(t.TempDir(), e, w, p, nil)

	// Reads pass authorization, so ListFiles must not report an access failure.
	// It legitimately returns an empty list because the workspace directory does
	// not exist, which is an fs result rather than a refusal.
	if _, err := fu.ListFiles(context.Background(), uuid.New(), viewer, "."); err != nil &&
		strings.Contains(err.Error(), "not found") {
		t.Errorf("a VIEWER should pass the read check, got %v", err)
	}
	if err := fu.DeleteFile(context.Background(), uuid.New(), viewer, "x"); err == nil ||
		!strings.Contains(err.Error(), "not found") {
		t.Errorf("a VIEWER must be refused a write, got %v", err)
	}
}
