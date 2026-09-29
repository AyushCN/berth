package domain

import "github.com/google/uuid"

// NATS subjects used to hand work from the api to the worker.
//
// The api publishes these; the worker subscribes. They used to be published
// with no subscriber at all, so every environment created over the HTTP API
// sat in CREATED forever with no container, no logs and no exec.
const (
	SubjectEnvironmentCreate = "berth.environment.create"
	SubjectEnvironmentStop   = "berth.environment.stop"
	SubjectEnvironmentStart  = "berth.environment.start"
	SubjectEnvironmentDelete = "berth.environment.delete"
)

// EnvironmentCreateEvent asks the worker to provision an environment.
//
// GitURL and GitBranch are also persisted on the workspace, which is the
// authoritative copy: this event is a latency optimisation, not the only
// source of truth. The worker should prefer reading the workspace and fall
// back to these fields only for disambiguation.
type EnvironmentCreateEvent struct {
	EnvironmentID uuid.UUID `json:"environment_id"`
	WorkspaceID   uuid.UUID `json:"workspace_id"`
	GitURL        string    `json:"git_url"`
	GitBranch     string    `json:"git_branch"`
	OwnerID       uuid.UUID `json:"owner_id"`
}

// EnvironmentLifecycleEvent asks the worker to stop or tear down the running
// container for an environment. Issued when the api has no Docker runtime of
// its own, which is the normal api-mode configuration.
type EnvironmentLifecycleEvent struct {
	EnvironmentID uuid.UUID `json:"environment_id"`
	WorkspaceID   uuid.UUID `json:"workspace_id,omitempty"`
	ContainerID   string    `json:"container_id"`
}
