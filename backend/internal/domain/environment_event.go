package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// EnvironmentEventType represents the type of environment event
type EnvironmentEventType string

const (
	// Lifecycle events
	EnvironmentEventStateChanged     EnvironmentEventType = "STATE_CHANGED"
	EnvironmentEventBuildStarted     EnvironmentEventType = "BUILD_STARTED"
	EnvironmentEventBuildFinished    EnvironmentEventType = "BUILD_FINISHED"
	EnvironmentEventBuildFailed      EnvironmentEventType = "BUILD_FAILED"
	EnvironmentEventContainerStarted EnvironmentEventType = "CONTAINER_STARTED"
	EnvironmentEventContainerStopped EnvironmentEventType = "CONTAINER_STOPPED"
	EnvironmentEventContainerCrashed EnvironmentEventType = "CONTAINER_CRASHED"
	EnvironmentEventSuspended        EnvironmentEventType = "SUSPENDED"
	EnvironmentEventResumed          EnvironmentEventType = "RESUMED"

	// File/Code events
	EnvironmentEventFileChanged EnvironmentEventType = "FILE_CHANGED"
	EnvironmentEventFileCreated EnvironmentEventType = "FILE_CREATED"
	EnvironmentEventFileDeleted EnvironmentEventType = "FILE_DELETED"
	EnvironmentEventGitCommit   EnvironmentEventType = "GIT_COMMIT"
	EnvironmentEventGitPush     EnvironmentEventType = "GIT_PUSH"
	EnvironmentEventGitPull     EnvironmentEventType = "GIT_PULL"

	// User presence events
	EnvironmentEventUserJoined  EnvironmentEventType = "USER_JOINED"
	EnvironmentEventUserLeft    EnvironmentEventType = "USER_LEFT"
	EnvironmentEventUserTyping  EnvironmentEventType = "USER_TYPING"
	EnvironmentEventCursorMoved EnvironmentEventType = "CURSOR_MOVED"

	// Collaboration events
	EnvironmentEventChangeRequestCreated EnvironmentEventType = "CHANGE_REQUEST_CREATED"
	EnvironmentEventChangeRequestUpdated EnvironmentEventType = "CHANGE_REQUEST_UPDATED"
	EnvironmentEventChangeRequestMerged  EnvironmentEventType = "CHANGE_REQUEST_MERGED"

	// Terminal events
	EnvironmentEventTerminalOutput EnvironmentEventType = "TERMINAL_OUTPUT"
	EnvironmentEventTerminalResize EnvironmentEventType = "TERMINAL_RESIZE"
)

// EnvironmentEvent represents an event in an environment/workspace/project
type EnvironmentEvent struct {
	ID            uuid.UUID            `json:"id"`
	EnvironmentID *uuid.UUID           `json:"environment_id,omitempty"`
	WorkspaceID   uuid.UUID            `json:"workspace_id"`
	ProjectID     uuid.UUID            `json:"project_id"`
	UserID        *uuid.UUID           `json:"user_id,omitempty"`
	Type          EnvironmentEventType `json:"type"`
	Payload       map[string]any       `json:"payload"`
	CreatedAt     time.Time            `json:"created_at"`
}

// EnvironmentEventRepository defines the interface for environment event persistence
type EnvironmentEventRepository interface {
	Create(ctx context.Context, event *EnvironmentEvent) error
	GetByEnvironment(ctx context.Context, environmentID uuid.UUID, limit, offset int) ([]*EnvironmentEvent, error)
	GetByWorkspace(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]*EnvironmentEvent, error)
	GetByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*EnvironmentEvent, error)
	GetRecent(ctx context.Context, projectID uuid.UUID, since time.Time, limit int) ([]*EnvironmentEvent, error)
}
