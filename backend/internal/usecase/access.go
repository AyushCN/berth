package usecase

import (
	"context"
	"fmt"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

type accessLevel int

const (
	accessNone accessLevel = iota
	accessRead
	accessWrite
)

// resolveAccess determines what a user may do with an environment's files and
// git history.
//
// The rules mirror EnvironmentUsecase: the workspace owner has full access, and
// anyone else needs to be a collaborator on the owning project. A VIEWER gets
// read-only; EDITOR and OWNER get write.
//
// This exists as one shared function because the two callers disagreed. The
// file endpoints performed no authorization whatsoever, so any authenticated
// user could read, overwrite or delete the files of any environment by UUID.
// GitUsecase.Authorize meanwhile compared ws.OwnerID directly, so a
// collaborator who could open the workspace was refused on every git call.
func resolveAccess(
	ctx context.Context,
	envRepo domain.EnvironmentRepository,
	wsRepo domain.WorkspaceRepository,
	projRepo domain.ProjectRepository,
	environmentID, userID uuid.UUID,
) (accessLevel, error) {
	env, err := envRepo.GetByID(ctx, environmentID)
	if err != nil {
		return accessNone, fmt.Errorf("environment not found")
	}

	ws, err := wsRepo.GetByID(ctx, env.WorkspaceID)
	if err != nil {
		return accessNone, fmt.Errorf("workspace not found")
	}

	if ws.OwnerID == userID {
		return accessWrite, nil
	}

	if ws.ProjectID == uuid.Nil {
		return accessNone, nil
	}

	collab, err := projRepo.GetCollaborator(ctx, ws.ProjectID, userID)
	if err != nil || collab == nil {
		return accessNone, nil
	}
	if collab.Role == domain.ProjectRoleViewer {
		return accessRead, nil
	}
	return accessWrite, nil
}

// requireAccess returns an error unless the user has at least the level given.
func requireAccess(ctx context.Context, envRepo domain.EnvironmentRepository, wsRepo domain.WorkspaceRepository, projRepo domain.ProjectRepository, environmentID, userID uuid.UUID, want accessLevel, action string) error {
	level, err := resolveAccess(ctx, envRepo, wsRepo, projRepo, environmentID, userID)
	if err != nil {
		return err
	}
	if level < want {
		// Deliberately "not found" rather than "forbidden": a 403 confirms the
		// environment exists, which lets an attacker enumerate UUIDs.
		return fmt.Errorf("environment not found")
	}
	_ = action
	return nil
}
