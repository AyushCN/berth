package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

// fakePublisher records published messages.
type fakePublisher struct {
	subject string
	payload []byte
	err     error
}

func (f *fakePublisher) Publish(subject string, data []byte) error {
	f.subject = subject
	f.payload = data
	return f.err
}

// fakeRuntime records container operations.
type fakeRuntime struct {
	started []string
	stopped []string
	err     error
}

func (f *fakeRuntime) Start(_ context.Context, id string) error {
	if f.err != nil {
		return f.err
	}
	f.started = append(f.started, id)
	return nil
}

func (f *fakeRuntime) Stop(_ context.Context, id string) error {
	if f.err != nil {
		return f.err
	}
	f.stopped = append(f.stopped, id)
	return nil
}

// The other ContainerRuntime methods are unused by the controllers.
func (f *fakeRuntime) Create(context.Context, domain.ContainerSpec) (string, error) {
	return "", errors.New("not implemented")
}
func (f *fakeRuntime) Remove(context.Context, string) error { return nil }
func (f *fakeRuntime) Exec(context.Context, string, []string) (string, error) {
	return "", nil
}
func (f *fakeRuntime) ExecWithEnv(context.Context, string, []string, map[string]string) (string, error) {
	return "", nil
}
func (f *fakeRuntime) CommitContainer(context.Context, string, string) error { return nil }
func (f *fakeRuntime) ExecPTY(context.Context, string, []string) (io.WriteCloser, io.Reader, func() error, error) {
	return nil, nil, nil, errors.New("not implemented")
}
func (f *fakeRuntime) GetLogs(context.Context, string, int) (string, error) { return "", nil }

func TestDockerContainerControl(t *testing.T) {
	rt := &fakeRuntime{}
	ctrl := NewDockerContainerControl(rt)
	env := &domain.Environment{ID: uuid.New(), ContainerID: "abc123"}

	if err := ctrl.Stop(context.Background(), env); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if err := ctrl.Start(context.Background(), env); err != nil {
		t.Fatalf("start: %v", err)
	}
	if len(rt.stopped) != 1 || rt.stopped[0] != "abc123" {
		t.Errorf("expected stop of abc123, got %v", rt.stopped)
	}
	if len(rt.started) != 1 || rt.started[0] != "abc123" {
		t.Errorf("expected start of abc123, got %v", rt.started)
	}
}

func TestDockerContainerControlNoContainerIsNoopForStop(t *testing.T) {
	rt := &fakeRuntime{}
	ctrl := NewDockerContainerControl(rt)
	// An environment with no container cannot be stopped, but that must not be
	// an error: there is nothing to do.
	if err := ctrl.Stop(context.Background(), &domain.Environment{ID: uuid.New()}); err != nil {
		t.Errorf("stop with no container should be a no-op, got %v", err)
	}
}

func TestDockerContainerControlStartWithoutContainerErrors(t *testing.T) {
	rt := &fakeRuntime{}
	ctrl := NewDockerContainerControl(rt)
	if err := ctrl.Start(context.Background(), &domain.Environment{ID: uuid.New()}); err == nil {
		t.Error("expected an error when there is no container to start")
	}
}

func TestNATSContainerControlPublishes(t *testing.T) {
	pub := &fakePublisher{}
	ctrl := NewNATSContainerControl(pub)
	env := &domain.Environment{ID: uuid.New(), ContainerID: "abc123"}

	if err := ctrl.Stop(context.Background(), env); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if pub.subject != domain.SubjectEnvironmentStop {
		t.Errorf("stop published to %q, want %q", pub.subject, domain.SubjectEnvironmentStop)
	}

	if err := ctrl.Start(context.Background(), env); err != nil {
		t.Fatalf("start: %v", err)
	}
	if pub.subject != domain.SubjectEnvironmentStart {
		t.Errorf("start published to %q, want %q", pub.subject, domain.SubjectEnvironmentStart)
	}

	var evt domain.EnvironmentLifecycleEvent
	if err := json.Unmarshal(pub.payload, &evt); err != nil {
		t.Fatalf("payload is not valid JSON: %v", err)
	}
	if evt.EnvironmentID != env.ID {
		t.Errorf("payload environment id %s, want %s", evt.EnvironmentID, env.ID)
	}
	if evt.ContainerID != "abc123" {
		t.Errorf("payload container id %q, want abc123", evt.ContainerID)
	}
}

// Regression: the api stored a nil domain.ContainerRuntime and then called a
// method on it, which panicked the whole process. A controller with no
// publisher must return an error instead.
func TestNATSContainerControlNilPublisherErrorsNotPanics(t *testing.T) {
	ctrl := NewNATSContainerControl(nil)
	env := &domain.Environment{ID: uuid.New(), ContainerID: "abc123"}

	if err := ctrl.Stop(context.Background(), env); err == nil {
		t.Error("expected stop to error with no publisher")
	}
	if err := ctrl.Start(context.Background(), env); err == nil {
		t.Error("expected start to error with no publisher")
	}
}

func TestNATSContainerControlPropagatesPublishError(t *testing.T) {
	pub := &fakePublisher{err: errors.New("nats down")}
	ctrl := NewNATSContainerControl(pub)
	if err := ctrl.Stop(context.Background(), &domain.Environment{ID: uuid.New()}); err == nil {
		t.Error("expected publish failure to surface")
	}
}

// NewActivityTracker must never accept a nil controller, because that was the
// original crash.
func TestNewActivityTrackerNeverHasNilControl(t *testing.T) {
	at := NewActivityTracker(nil, nil, 0, 0)
	if at.control == nil {
		t.Fatal("tracker built with a nil controller")
	}
	// And using it must error, not panic.
	err := at.control.Stop(context.Background(), &domain.Environment{ID: uuid.New(), ContainerID: "x"})
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "no message bus") {
		t.Errorf("unexpected error: %v", err)
	}
}
