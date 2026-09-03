package operations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QihuiPan/internal-developer-platform/internal/domain"
	"github.com/QihuiPan/internal-developer-platform/internal/store"
)

func testDescriptor() domain.ServiceDescriptor {
	return domain.ServiceDescriptor{APIVersion: "platform.demo/v1", Kind: "Service", Metadata: domain.Metadata{Name: "payments-notifier", Owner: "team-payments"}, Spec: domain.ServiceSpec{Template: "go-http@1.0.0", Runtime: domain.RuntimeSpec{Port: 8080, Replicas: 2}, Resources: []domain.ResourceSpec{{Type: "postgres", Plan: "small"}}, Environments: []string{"dev", "staging"}, Observability: domain.ObservabilitySpec{AvailabilitySLO: 99.9}}}
}

func TestProcessGeneratesADeployableRepository(t *testing.T) {
	root := t.TempDir()
	state, err := store.Open(filepath.Join(root, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	created, err := state.CreateService("alice", "test", "request-123", testDescriptor())
	if err != nil {
		t.Fatal(err)
	}
	processor := NewProcessor(state, filepath.Join(root, "generated"), "")
	if err := processor.Process(created.Operation.ID); err != nil {
		t.Fatal(err)
	}
	operation, err := state.Operation(created.Operation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if operation.Status != domain.OperationSucceeded {
		t.Fatalf("unexpected status: %s", operation.Status)
	}
	for _, relative := range []string{"README.md", "service.yaml", "cmd/server/main.go", "Dockerfile", "deploy/kubernetes.yaml"} {
		if _, err := os.Stat(filepath.Join(root, "generated", "payments-notifier", filepath.FromSlash(relative))); err != nil {
			t.Errorf("missing %s: %v", relative, err)
		}
	}
}

func TestProcessSupportsEveryPublishedTemplate(t *testing.T) {
	for _, template := range []string{"go-http@1.0.0", "python-http@1.0.0", "node-http@1.0.0"} {
		t.Run(template, func(t *testing.T) {
			root := t.TempDir()
			state, err := store.Open(filepath.Join(root, "state.json"))
			if err != nil {
				t.Fatal(err)
			}
			descriptor := testDescriptor()
			descriptor.Metadata.Name = strings.ReplaceAll(strings.Split(template, "@")[0], "-http", "-service")
			descriptor.Spec.Template = template
			created, err := state.CreateService("alice", "test", "request-123", descriptor)
			if err != nil {
				t.Fatal(err)
			}
			if err := NewProcessor(state, filepath.Join(root, "generated"), "").Process(created.Operation.ID); err != nil {
				t.Fatal(err)
			}
			operation, _ := state.Operation(created.Operation.ID)
			if operation.Status != domain.OperationSucceeded {
				t.Fatalf("unexpected status: %s", operation.Status)
			}
		})
	}
}

func TestFailureCanBeRetriedFromCheckpoint(t *testing.T) {
	root := t.TempDir()
	state, err := store.Open(filepath.Join(root, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	created, err := state.CreateService("alice", "test", "request-123", testDescriptor())
	if err != nil {
		t.Fatal(err)
	}
	failing := NewProcessor(state, filepath.Join(root, "generated"), "render")
	if err := failing.Process(created.Operation.ID); err == nil {
		t.Fatal("fault injection did not fail")
	}
	failed, _ := state.Operation(created.Operation.ID)
	if failed.Status != domain.OperationFailed {
		t.Fatalf("unexpected failed status: %s", failed.Status)
	}
	if _, err := state.RetryOperation(created.Operation.ID, "alice", "fault removed"); err != nil {
		t.Fatal(err)
	}
	recovered := NewProcessor(state, filepath.Join(root, "generated"), "")
	if err := recovered.Process(created.Operation.ID); err != nil {
		t.Fatal(err)
	}
	completed, _ := state.Operation(created.Operation.ID)
	if completed.Status != domain.OperationSucceeded {
		t.Fatalf("retry did not recover: %s", completed.Status)
	}
}
