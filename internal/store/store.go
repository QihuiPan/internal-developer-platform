package store

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/QihuiPan/internal-developer-platform/internal/domain"
)

var ErrNotFound = errors.New("record not found")
var ErrConflict = errors.New("record already exists")
var ErrIdempotencyConflict = errors.New("idempotency key was already used for a different request")

type persistedState struct {
	Services    map[string]domain.ServiceRecord `json:"services"`
	Operations  map[string]domain.Operation     `json:"operations"`
	Idempotency map[string]string               `json:"idempotency"`
	AuditEvents []domain.AuditEvent             `json:"auditEvents"`
}

type Store struct {
	mu    sync.RWMutex
	path  string
	state persistedState
}

func Open(path string) (*Store, error) {
	s := &Store{path: path, state: persistedState{
		Services: map[string]domain.ServiceRecord{}, Operations: map[string]domain.Operation{}, Idempotency: map[string]string{}, AuditEvents: []domain.AuditEvent{},
	}}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read state: %w", err)
	}
	if err := json.Unmarshal(data, &s.state); err != nil {
		return nil, fmt.Errorf("decode state: %w", err)
	}
	s.ensureMaps()
	return s, nil
}

func (s *Store) ensureMaps() {
	if s.state.Services == nil {
		s.state.Services = map[string]domain.ServiceRecord{}
	}
	if s.state.Operations == nil {
		s.state.Operations = map[string]domain.Operation{}
	}
	if s.state.Idempotency == nil {
		s.state.Idempotency = map[string]string{}
	}
	if s.state.AuditEvents == nil {
		s.state.AuditEvents = []domain.AuditEvent{}
	}
}

func (s *Store) CreateService(actor, reason, key string, descriptor domain.ServiceDescriptor) (domain.CreateResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	requestData, err := json.Marshal(descriptor)
	if err != nil {
		return domain.CreateResult{}, fmt.Errorf("encode request fingerprint: %w", err)
	}
	requestHash := fmt.Sprintf("%x", sha256.Sum256(requestData))
	if operationID, ok := s.state.Idempotency[key]; ok {
		operation := s.state.Operations[operationID]
		if operation.RequestHash != requestHash {
			return domain.CreateResult{}, ErrIdempotencyConflict
		}
		service := s.state.Services[operation.Target]
		return domain.CreateResult{Service: service, Operation: operation, Replayed: true}, nil
	}
	if _, ok := s.state.Services[descriptor.Metadata.Name]; ok {
		return domain.CreateResult{}, ErrConflict
	}
	now := time.Now().UTC()
	operationID, err := newID("op")
	if err != nil {
		return domain.CreateResult{}, err
	}
	service := domain.ServiceRecord{Descriptor: descriptor, Status: domain.ServiceProvisioning, CreatedAt: now, UpdatedAt: now}
	operation := domain.Operation{
		ID: operationID, Type: "CREATE_SERVICE", Target: descriptor.Metadata.Name, IdempotencyKey: key, RequestHash: requestHash,
		Status: domain.OperationPending, Attempt: 1, CreatedAt: now, UpdatedAt: now,
		Steps: []domain.OperationStep{{Name: "validate", Status: domain.StepPending}, {Name: "plan", Status: domain.StepPending}, {Name: "render", Status: domain.StepPending}, {Name: "verify", Status: domain.StepPending}},
	}
	auditID, err := newID("audit")
	if err != nil {
		return domain.CreateResult{}, err
	}
	s.state.Services[descriptor.Metadata.Name] = service
	s.state.Operations[operationID] = operation
	s.state.Idempotency[key] = operationID
	s.state.AuditEvents = append(s.state.AuditEvents, domain.AuditEvent{ID: auditID, Actor: actor, Action: "service.create.requested", Target: descriptor.Metadata.Name, Reason: reason, After: descriptor, CreatedAt: now})
	if err := s.saveLocked(); err != nil {
		return domain.CreateResult{}, err
	}
	return domain.CreateResult{Service: service, Operation: operation}, nil
}

func (s *Store) Service(name string) (domain.ServiceRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.state.Services[name]
	if !ok {
		return domain.ServiceRecord{}, ErrNotFound
	}
	return value, nil
}

func (s *Store) Operation(id string) (domain.Operation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.state.Operations[id]
	if !ok {
		return domain.Operation{}, ErrNotFound
	}
	return value, nil
}

func (s *Store) PendingOperations() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := []string{}
	for id, operation := range s.state.Operations {
		if operation.Status == domain.OperationPending || operation.Status == domain.OperationRetrying {
			ids = append(ids, id)
		}
	}
	return ids
}

func (s *Store) StartStep(operationID, name string, status domain.OperationStatus) (domain.Operation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	operation, ok := s.state.Operations[operationID]
	if !ok {
		return domain.Operation{}, ErrNotFound
	}
	now := time.Now().UTC()
	operation.Status, operation.UpdatedAt, operation.Error = status, now, ""
	for i := range operation.Steps {
		if operation.Steps[i].Name == name {
			operation.Steps[i].Status = domain.StepRunning
			operation.Steps[i].Attempt++
			operation.Steps[i].StartedAt = &now
			operation.Steps[i].FinishedAt = nil
			operation.Steps[i].Error = ""
		}
	}
	s.state.Operations[operationID] = operation
	if err := s.saveLocked(); err != nil {
		return domain.Operation{}, err
	}
	return operation, nil
}

func (s *Store) CompleteStep(operationID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	operation, ok := s.state.Operations[operationID]
	if !ok {
		return ErrNotFound
	}
	now := time.Now().UTC()
	for i := range operation.Steps {
		if operation.Steps[i].Name == name {
			operation.Steps[i].Status = domain.StepSucceeded
			operation.Steps[i].FinishedAt = &now
		}
	}
	operation.UpdatedAt = now
	s.state.Operations[operationID] = operation
	return s.saveLocked()
}

func (s *Store) FailOperation(operationID, stepName, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	operation, ok := s.state.Operations[operationID]
	if !ok {
		return ErrNotFound
	}
	now := time.Now().UTC()
	operation.Status, operation.Error, operation.UpdatedAt = domain.OperationFailed, message, now
	for i := range operation.Steps {
		if operation.Steps[i].Name == stepName {
			operation.Steps[i].Status, operation.Steps[i].Error, operation.Steps[i].FinishedAt = domain.StepFailed, message, &now
		}
	}
	service := s.state.Services[operation.Target]
	service.Status, service.UpdatedAt = domain.ServiceFailed, now
	s.state.Services[operation.Target] = service
	s.state.Operations[operationID] = operation
	return s.saveLocked()
}

func (s *Store) SucceedOperation(operationID string, links map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	operation, ok := s.state.Operations[operationID]
	if !ok {
		return ErrNotFound
	}
	now := time.Now().UTC()
	operation.Status, operation.Links, operation.UpdatedAt = domain.OperationSucceeded, links, now
	service := s.state.Services[operation.Target]
	service.Status, service.UpdatedAt = domain.ServiceReady, now
	service.RepositoryURL, service.GitOpsPath, service.DashboardURL = links["repository"], links["gitops"], links["dashboard"]
	s.state.Services[operation.Target] = service
	s.state.Operations[operationID] = operation
	auditID, err := newID("audit")
	if err != nil {
		return err
	}
	s.state.AuditEvents = append(s.state.AuditEvents, domain.AuditEvent{ID: auditID, Actor: "platform-worker", Action: "service.create.succeeded", Target: operation.Target, Reason: "All reconciliation steps completed", After: links, CreatedAt: now})
	return s.saveLocked()
}

func (s *Store) RetryOperation(id, actor, reason string) (domain.Operation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	operation, ok := s.state.Operations[id]
	if !ok {
		return domain.Operation{}, ErrNotFound
	}
	if operation.Status != domain.OperationFailed {
		return domain.Operation{}, fmt.Errorf("only failed operations can be retried")
	}
	now := time.Now().UTC()
	operation.Status, operation.Error, operation.UpdatedAt = domain.OperationRetrying, "", now
	operation.Attempt++
	for i := range operation.Steps {
		if operation.Steps[i].Status == domain.StepFailed {
			operation.Steps[i].Status, operation.Steps[i].Error = domain.StepPending, ""
		}
	}
	service := s.state.Services[operation.Target]
	service.Status, service.UpdatedAt = domain.ServiceProvisioning, now
	s.state.Services[operation.Target], s.state.Operations[id] = service, operation
	auditID, err := newID("audit")
	if err != nil {
		return domain.Operation{}, err
	}
	s.state.AuditEvents = append(s.state.AuditEvents, domain.AuditEvent{ID: auditID, Actor: actor, Action: "operation.retry", Target: id, Reason: reason, CreatedAt: now})
	if err := s.saveLocked(); err != nil {
		return domain.Operation{}, err
	}
	return operation, nil
}

func (s *Store) AuditEvents() []domain.AuditEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]domain.AuditEvent(nil), s.state.AuditEvents...)
}

func (s *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o750); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	temporary := s.path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return fmt.Errorf("write state: %w", err)
	}
	if err := os.Rename(temporary, s.path); err != nil {
		return fmt.Errorf("replace state: %w", err)
	}
	return nil
}

func newID(prefix string) (string, error) {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate identifier: %w", err)
	}
	return prefix + "_" + hex.EncodeToString(buffer), nil
}
