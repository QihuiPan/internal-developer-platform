package store

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/QihuiPan/internal-developer-platform/internal/domain"
)

func descriptor(name string) domain.ServiceDescriptor {
	return domain.ServiceDescriptor{APIVersion: "platform.demo/v1", Kind: "Service", Metadata: domain.Metadata{Name: name, Owner: "team-payments"}, Spec: domain.ServiceSpec{Template: "go-http@1.0.0", Runtime: domain.RuntimeSpec{Port: 8080, Replicas: 2}, Environments: []string{"dev"}, Observability: domain.ObservabilitySpec{AvailabilitySLO: 99.9}}}
}

func TestCreateServiceIsDurableAndIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	state, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	first, err := state.CreateService("alice", "test", "request-123", descriptor("payments-notifier"))
	if err != nil {
		t.Fatal(err)
	}
	replay, err := state.CreateService("alice", "test", "request-123", descriptor("payments-notifier"))
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Replayed || replay.Operation.ID != first.Operation.ID {
		t.Fatal("duplicate request did not replay the original operation")
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	operation, err := reopened.Operation(first.Operation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if operation.Target != "payments-notifier" {
		t.Fatalf("unexpected target: %s", operation.Target)
	}
}

func TestIdempotencyKeyRejectsDifferentPayload(t *testing.T) {
	state, err := Open(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := state.CreateService("alice", "test", "request-123", descriptor("payments-notifier")); err != nil {
		t.Fatal(err)
	}
	_, err = state.CreateService("alice", "test", "request-123", descriptor("reporting-worker"))
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
}

func TestListsServicesAndOperationsNewestFirst(t *testing.T) {
	state, err := Open(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	for index, name := range []string{"first-service", "second-service"} {
		if _, err := state.CreateService("alice", "test", fmt.Sprintf("request-%03d", index), descriptor(name)); err != nil {
			t.Fatal(err)
		}
	}
	services := state.Services()
	operations := state.Operations()
	if len(services) != 2 || len(operations) != 2 {
		t.Fatalf("unexpected list lengths: services=%d operations=%d", len(services), len(operations))
	}
	if services[0].Descriptor.Metadata.Name != "second-service" || operations[0].Target != "second-service" {
		t.Fatal("lists are not ordered newest first")
	}
}
