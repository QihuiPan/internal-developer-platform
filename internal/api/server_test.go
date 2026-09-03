package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/QihuiPan/internal-developer-platform/internal/domain"
	"github.com/QihuiPan/internal-developer-platform/internal/operations"
	"github.com/QihuiPan/internal-developer-platform/internal/store"
)

func TestCreateServiceEndToEnd(t *testing.T) {
	root := t.TempDir()
	state, err := store.Open(filepath.Join(root, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	processor := operations.NewProcessor(state, filepath.Join(root, "generated"), "")
	processor.Start()
	defer processor.Stop()
	handler := NewServer(state, processor, slog.New(slog.NewTextHandler(io.Discard, nil)))
	descriptor := domain.ServiceDescriptor{APIVersion: "platform.demo/v1", Kind: "Service", Metadata: domain.Metadata{Name: "payments-notifier", Owner: "team-payments"}, Spec: domain.ServiceSpec{Template: "go-http@1.0.0", Runtime: domain.RuntimeSpec{Port: 8080, Replicas: 2}, Environments: []string{"dev"}, Observability: domain.ObservabilitySpec{AvailabilitySLO: 99.9}}}
	body, _ := json.Marshal(descriptor)
	request := httptest.NewRequest(http.MethodPost, "/v1/services", bytes.NewReader(body))
	request.Header.Set("Idempotency-Key", "request-123")
	request.Header.Set("X-Actor", "alice")
	request.Header.Set("X-Role", "developer")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("unexpected status %d: %s", response.Code, response.Body.String())
	}
	var result domain.CreateResult
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		operation, err := state.Operation(result.Operation.ID)
		if err != nil {
			t.Fatal(err)
		}
		if operation.Status == domain.OperationSucceeded {
			return
		}
		if operation.Status == domain.OperationFailed {
			t.Fatalf("operation failed: %s", operation.Error)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("operation did not complete before the test deadline")
}

func TestCreateServiceRequiresAuthentication(t *testing.T) {
	root := t.TempDir()
	state, err := store.Open(filepath.Join(root, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	processor := operations.NewProcessor(state, filepath.Join(root, "generated"), "")
	handler := NewServer(state, processor, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodPost, "/v1/services", bytes.NewReader([]byte("{}")))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: %d", response.Code)
	}
}
