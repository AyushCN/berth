package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/AyushCN/berth/internal/domain"
)

// containerControl abstracts "stop or start this environment's container".
//
// This exists because the api and the worker do not have the same abilities.
// The worker owns the Docker socket and can act directly. The api has no
// runtime in normal operation and must ask the worker over NATS.
//
// It replaces a bare domain.ContainerRuntime field that was set to nil in api
// mode and then dereferenced, panicking the whole api process the first time an
// idle environment was found.
type containerControl interface {
	Stop(ctx context.Context, env *domain.Environment) error
	Start(ctx context.Context, env *domain.Environment) error
}

// eventPublisher is the subset of the NATS client these controllers need.
type eventPublisher interface {
	Publish(subject string, data []byte) error
}

// dockerContainerControl acts directly through a container runtime. Used by the
// worker, which holds the Docker socket.
type dockerContainerControl struct {
	runtime domain.ContainerRuntime
}

func (c dockerContainerControl) Stop(ctx context.Context, env *domain.Environment) error {
	if env.ContainerID == "" {
		return nil
	}
	return c.runtime.StopSandbox(ctx, env.ContainerID)
}

func (c dockerContainerControl) Start(ctx context.Context, env *domain.Environment) error {
	if env.ContainerID == "" {
		return fmt.Errorf("environment has no container to start")
	}
	return c.runtime.StartSandbox(ctx, env.ContainerID)
}

// natsContainerControl asks the worker to act. Used by the api, which has no
// runtime of its own.
type natsContainerControl struct {
	publisher eventPublisher
}

func (c natsContainerControl) request(ctx context.Context, subject string, env *domain.Environment, action string) error {
	if c.publisher == nil {
		return fmt.Errorf("cannot %s environment %s: no message bus configured", action, env.ID)
	}
	payload, err := json.Marshal(domain.EnvironmentLifecycleEvent{
		EnvironmentID: env.ID,
		ContainerID:   env.ContainerID,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal %s request: %w", action, err)
	}
	if err := c.publisher.Publish(subject, payload); err != nil {
		return fmt.Errorf("failed to request environment %s: %w", action, err)
	}
	slog.Info("requested environment action from worker", "action", action, "environment_id", env.ID)
	return nil
}

func (c natsContainerControl) Stop(ctx context.Context, env *domain.Environment) error {
	return c.request(ctx, domain.SubjectEnvironmentStop, env, "stop")
}

func (c natsContainerControl) Start(ctx context.Context, env *domain.Environment) error {
	return c.request(ctx, domain.SubjectEnvironmentStart, env, "start")
}

// NewDockerContainerControl returns a controller that acts through runtime.
func NewDockerContainerControl(runtime domain.ContainerRuntime) containerControl {
	return dockerContainerControl{runtime: runtime}
}

// NewNATSContainerControl returns a controller that asks the worker over NATS.
// A nil publisher is tolerated and surfaces as an error on use rather than a
// nil dereference.
func NewNATSContainerControl(publisher eventPublisher) containerControl {
	return natsContainerControl{publisher: publisher}
}
