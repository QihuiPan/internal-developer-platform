package api

import (
	"archive/zip"
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/QihuiPan/internal-developer-platform/internal/domain"
	"github.com/QihuiPan/internal-developer-platform/internal/operations"
	"github.com/QihuiPan/internal-developer-platform/internal/store"
)

type Server struct {
	store     *store.Store
	processor *operations.Processor
	logger    *slog.Logger
	config    Config
	requests  atomic.Uint64
	errors    atomic.Uint64
}

type Config struct {
	AuthMode      string
	Token         string
	TokenRole     domain.Role
	Version       string
	GeneratedRoot string
	UI            http.Handler
}

type identity struct {
	Actor string
	Role  domain.Role
}

type errorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
}

func NewServer(state *store.Store, processor *operations.Processor, logger *slog.Logger) http.Handler {
	return NewServerWithConfig(state, processor, logger, Config{AuthMode: "demo", Version: "dev"})
}

func NewServerWithConfig(state *store.Store, processor *operations.Processor, logger *slog.Logger, config Config) http.Handler {
	if config.AuthMode == "" {
		config.AuthMode = "demo"
	}
	if config.TokenRole == "" {
		config.TokenRole = domain.RolePlatformAdmin
	}
	if config.Version == "" {
		config.Version = "dev"
	}
	if config.GeneratedRoot == "" {
		config.GeneratedRoot = filepath.Join(".platform", "generated")
	}
	server := &Server{store: state, processor: processor, logger: logger, config: config}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", server.health)
	mux.HandleFunc("GET /readyz", server.ready)
	mux.HandleFunc("GET /metrics", server.metrics)
	mux.HandleFunc("GET /v1/config", server.getConfig)
	mux.HandleFunc("GET /v1/services", server.listServices)
	mux.HandleFunc("POST /v1/services", server.createService)
	mux.HandleFunc("GET /v1/services/{name}/archive", server.downloadService)
	mux.HandleFunc("GET /v1/services/{name}", server.getService)
	mux.HandleFunc("GET /v1/operations", server.listOperations)
	mux.HandleFunc("GET /v1/operations/{id}", server.getOperation)
	mux.HandleFunc("POST /v1/operations/{id}/retry", server.retryOperation)
	mux.HandleFunc("GET /v1/audit-events", server.getAuditEvents)
	if config.UI != nil {
		mux.Handle("GET /", config.UI)
	}
	return server.observe(mux)
}

func (s *Server) observe(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" {
			requestID = fmt.Sprintf("req-%d", started.UnixNano())
		}
		w.Header().Set("X-Request-ID", requestID)
		s.requests.Add(1)
		next.ServeHTTP(w, r)
		s.logger.Info("request completed", "request_id", requestID, "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(started).Milliseconds())
	})
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
func (s *Server) ready(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) getConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"authMode": s.config.AuthMode,
		"version":  s.config.Version,
	})
}

func (s *Server) metrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = fmt.Fprintf(w, "# HELP platform_http_requests_total HTTP requests received.\n# TYPE platform_http_requests_total counter\nplatform_http_requests_total %d\n# HELP platform_http_errors_total HTTP errors returned.\n# TYPE platform_http_errors_total counter\nplatform_http_errors_total %d\n", s.requests.Load(), s.errors.Load())
}

func (s *Server) createService(w http.ResponseWriter, r *http.Request) {
	caller, ok := s.authorize(w, r, "service:create")
	if !ok {
		return
	}
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if len(key) < 8 || len(key) > 128 {
		s.fail(w, r, http.StatusBadRequest, "INVALID_IDEMPOTENCY_KEY", "Idempotency-Key must contain 8 to 128 characters")
		return
	}
	var descriptor domain.ServiceDescriptor
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&descriptor); err != nil {
		s.fail(w, r, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	if err := domain.ValidateDescriptor(descriptor); err != nil {
		s.fail(w, r, http.StatusUnprocessableEntity, "VALIDATION_FAILED", err.Error())
		return
	}
	reason := strings.TrimSpace(r.Header.Get("X-Reason"))
	if reason == "" {
		reason = "Self-service create request"
	}
	result, err := s.store.CreateService(caller.Actor, reason, key, descriptor)
	if errors.Is(err, store.ErrIdempotencyConflict) {
		s.fail(w, r, http.StatusConflict, "IDEMPOTENCY_CONFLICT", err.Error())
		return
	}
	if errors.Is(err, store.ErrConflict) {
		s.fail(w, r, http.StatusConflict, "SERVICE_EXISTS", "A service with this name already exists")
		return
	}
	if err != nil {
		s.fail(w, r, http.StatusInternalServerError, "STORE_ERROR", err.Error())
		return
	}
	if !result.Replayed {
		s.processor.Enqueue(result.Operation.ID)
	}
	status := http.StatusAccepted
	if result.Replayed {
		status = http.StatusOK
	}
	w.Header().Set("Location", "/v1/operations/"+result.Operation.ID)
	writeJSON(w, status, result)
}

func (s *Server) listServices(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.authorize(w, r, "service:read"); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": s.store.Services()})
}

func (s *Server) getService(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.authorize(w, r, "service:read"); !ok {
		return
	}
	service, err := s.store.Service(r.PathValue("name"))
	if errors.Is(err, store.ErrNotFound) {
		s.fail(w, r, http.StatusNotFound, "NOT_FOUND", "Service not found")
		return
	}
	if err != nil {
		s.fail(w, r, http.StatusInternalServerError, "STORE_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, service)
}

func (s *Server) downloadService(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.authorize(w, r, "service:read"); !ok {
		return
	}
	service, err := s.store.Service(r.PathValue("name"))
	if errors.Is(err, store.ErrNotFound) {
		s.fail(w, r, http.StatusNotFound, "NOT_FOUND", "Service not found")
		return
	}
	if err != nil {
		s.fail(w, r, http.StatusInternalServerError, "STORE_ERROR", err.Error())
		return
	}
	if service.Status != domain.ServiceReady {
		s.fail(w, r, http.StatusConflict, "ARTIFACT_NOT_READY", "The generated repository is not ready")
		return
	}
	name := service.Descriptor.Metadata.Name
	archive, err := archiveDirectory(filepath.Join(s.config.GeneratedRoot, name), name)
	if err != nil {
		s.fail(w, r, http.StatusInternalServerError, "ARCHIVE_ERROR", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.zip"`, name))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(archive)
}

func archiveDirectory(root, prefix string) ([]byte, error) {
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !entry.Type().IsRegular() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		writer, err := archive.Create(filepath.ToSlash(filepath.Join(prefix, relative)))
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = writer.Write(data)
		return err
	})
	if err != nil {
		_ = archive.Close()
		return nil, fmt.Errorf("archive generated repository: %w", err)
	}
	if err := archive.Close(); err != nil {
		return nil, fmt.Errorf("finish generated repository archive: %w", err)
	}
	return buffer.Bytes(), nil
}

func (s *Server) getOperation(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.authorize(w, r, "operation:read"); !ok {
		return
	}
	operation, err := s.store.Operation(r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		s.fail(w, r, http.StatusNotFound, "NOT_FOUND", "Operation not found")
		return
	}
	if err != nil {
		s.fail(w, r, http.StatusInternalServerError, "STORE_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, operation)
}

func (s *Server) listOperations(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.authorize(w, r, "operation:read"); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": s.store.Operations()})
}

func (s *Server) retryOperation(w http.ResponseWriter, r *http.Request) {
	caller, ok := s.authorize(w, r, "operation:retry")
	if !ok {
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&body); err != nil {
		s.fail(w, r, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	if strings.TrimSpace(body.Reason) == "" {
		s.fail(w, r, http.StatusBadRequest, "REASON_REQUIRED", "A retry reason is required")
		return
	}
	operation, err := s.store.RetryOperation(r.PathValue("id"), caller.Actor, body.Reason)
	if errors.Is(err, store.ErrNotFound) {
		s.fail(w, r, http.StatusNotFound, "NOT_FOUND", "Operation not found")
		return
	}
	if err != nil {
		s.fail(w, r, http.StatusConflict, "NOT_RETRYABLE", err.Error())
		return
	}
	s.processor.Enqueue(operation.ID)
	writeJSON(w, http.StatusAccepted, operation)
}

func (s *Server) getAuditEvents(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.authorize(w, r, "audit:read"); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": s.store.AuditEvents()})
}

func (s *Server) authorize(w http.ResponseWriter, r *http.Request, action string) (identity, bool) {
	caller := identity{Actor: strings.TrimSpace(r.Header.Get("X-Actor"))}
	switch s.config.AuthMode {
	case "demo":
		caller.Role = domain.Role(strings.TrimSpace(r.Header.Get("X-Role")))
		if caller.Actor == "" || caller.Role == "" {
			s.fail(w, r, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "X-Actor and X-Role headers are required in demo authentication mode")
			return identity{}, false
		}
	case "token":
		if len(s.config.Token) < 16 || !domain.ValidRole(s.config.TokenRole) {
			s.fail(w, r, http.StatusInternalServerError, "AUTH_CONFIGURATION_ERROR", "Token authentication is not configured safely")
			return identity{}, false
		}
		authorization := r.Header.Get("Authorization")
		provided := strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
		if !strings.HasPrefix(authorization, "Bearer ") || subtle.ConstantTimeCompare([]byte(provided), []byte(s.config.Token)) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="platform-api"`)
			s.fail(w, r, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "A valid Bearer token is required")
			return identity{}, false
		}
		if caller.Actor == "" {
			s.fail(w, r, http.StatusUnauthorized, "ACTOR_REQUIRED", "X-Actor is required for audit attribution")
			return identity{}, false
		}
		caller.Role = s.config.TokenRole
	default:
		s.fail(w, r, http.StatusInternalServerError, "AUTH_CONFIGURATION_ERROR", "The server authentication mode is invalid")
		return identity{}, false
	}
	if !domain.Can(caller.Role, action) {
		s.fail(w, r, http.StatusForbidden, "FORBIDDEN", "The caller role is not allowed to perform this action")
		return identity{}, false
	}
	return caller, true
}

func (s *Server) fail(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	s.errors.Add(1)
	writeJSON(w, status, errorResponse{Code: code, Message: message, RequestID: w.Header().Get("X-Request-ID")})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
