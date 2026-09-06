package api

import (
	"archive/zip"
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

func TestTokenModeRejectsInvalidTokenAndUsesConfiguredRole(t *testing.T) {
	root := t.TempDir()
	state, err := store.Open(filepath.Join(root, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	processor := operations.NewProcessor(state, filepath.Join(root, "generated"), "")
	handler := NewServerWithConfig(state, processor, slog.New(slog.NewTextHandler(io.Discard, nil)), Config{
		AuthMode: "token", Token: "test-token-with-enough-length", TokenRole: domain.RoleAuditor,
	})

	unauthorized := httptest.NewRequest(http.MethodGet, "/v1/services", nil)
	unauthorized.Header.Set("X-Actor", "alice")
	unauthorizedResponse := httptest.NewRecorder()
	handler.ServeHTTP(unauthorizedResponse, unauthorized)
	if unauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected missing-token status: %d", unauthorizedResponse.Code)
	}
	malformed := httptest.NewRequest(http.MethodGet, "/v1/services", nil)
	malformed.Header.Set("X-Actor", "alice")
	malformed.Header.Set("Authorization", "test-token-with-enough-length")
	malformedResponse := httptest.NewRecorder()
	handler.ServeHTTP(malformedResponse, malformed)
	if malformedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected malformed-token status: %d", malformedResponse.Code)
	}

	authorized := httptest.NewRequest(http.MethodGet, "/v1/audit-events", nil)
	authorized.Header.Set("X-Actor", "alice")
	authorized.Header.Set("Authorization", "Bearer test-token-with-enough-length")
	authorizedResponse := httptest.NewRecorder()
	handler.ServeHTTP(authorizedResponse, authorized)
	if authorizedResponse.Code != http.StatusOK {
		t.Fatalf("unexpected configured-role status: %d", authorizedResponse.Code)
	}
}

func TestTokenModeFailsClosedWithUnsafeServerConfiguration(t *testing.T) {
	root := t.TempDir()
	state, err := store.Open(filepath.Join(root, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	processor := operations.NewProcessor(state, filepath.Join(root, "generated"), "")
	handler := NewServerWithConfig(state, processor, slog.New(slog.NewTextHandler(io.Discard, nil)), Config{AuthMode: "token", Token: "short"})
	request := httptest.NewRequest(http.MethodGet, "/v1/services", nil)
	request.Header.Set("X-Actor", "alice")
	request.Header.Set("Authorization", "Bearer short")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("unsafe token configuration did not fail closed: %d", response.Code)
	}
}

func TestListEndpointsReturnCollections(t *testing.T) {
	root := t.TempDir()
	state, err := store.Open(filepath.Join(root, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	processor := operations.NewProcessor(state, filepath.Join(root, "generated"), "")
	handler := NewServer(state, processor, slog.New(slog.NewTextHandler(io.Discard, nil)))
	for _, path := range []string{"/v1/services", "/v1/operations"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.Header.Set("X-Actor", "alice")
		request.Header.Set("X-Role", "developer")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"items":[]`)) {
			t.Fatalf("unexpected response for %s: %d %s", path, response.Code, response.Body.String())
		}
	}
}

func TestDownloadServiceReturnsGeneratedRepositoryArchive(t *testing.T) {
	root := t.TempDir()
	generatedRoot := filepath.Join(root, "generated")
	state, err := store.Open(filepath.Join(root, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	descriptor := domain.ServiceDescriptor{APIVersion: "platform.demo/v1", Kind: "Service", Metadata: domain.Metadata{Name: "payments-notifier", Owner: "team-payments"}, Spec: domain.ServiceSpec{Template: "go-http@1.0.0", Runtime: domain.RuntimeSpec{Port: 8080, Replicas: 2}, Environments: []string{"dev"}, Observability: domain.ObservabilitySpec{AvailabilitySLO: 99.9}}}
	created, err := state.CreateService("alice", "test", "download-request", descriptor)
	if err != nil {
		t.Fatal(err)
	}
	processor := operations.NewProcessor(state, generatedRoot, "")
	if err := processor.Process(created.Operation.ID); err != nil {
		t.Fatal(err)
	}
	handler := NewServerWithConfig(state, processor, slog.New(slog.NewTextHandler(io.Discard, nil)), Config{AuthMode: "demo", GeneratedRoot: generatedRoot})
	request := httptest.NewRequest(http.MethodGet, "/v1/services/payments-notifier/archive", nil)
	request.Header.Set("X-Actor", "alice")
	request.Header.Set("X-Role", "developer")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("unexpected archive response: %d %s", response.Code, response.Body.String())
	}
	archive, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, file := range archive.File {
		if file.Name == "payments-notifier/Dockerfile" {
			found = true
		}
	}
	if !found {
		t.Fatal("archive does not contain the generated Dockerfile")
	}
}
