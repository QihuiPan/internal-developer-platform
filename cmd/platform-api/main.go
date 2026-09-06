package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/QihuiPan/internal-developer-platform/internal/api"
	"github.com/QihuiPan/internal-developer-platform/internal/domain"
	"github.com/QihuiPan/internal-developer-platform/internal/operations"
	"github.com/QihuiPan/internal-developer-platform/internal/store"
	platformweb "github.com/QihuiPan/internal-developer-platform/web"
)

var version = "dev"
var commit = "none"
var buildDate = "unknown"

func main() {
	address := flag.String("address", environment("PLATFORM_ADDRESS", "127.0.0.1:8080"), "HTTP listen address")
	dataPath := flag.String("data", environment("PLATFORM_DATA_PATH", filepath.Join(".platform", "state.json")), "Persistent state file")
	generatedRoot := flag.String("generated-root", environment("GENERATED_SERVICES_DIR", filepath.Join(".platform", "generated")), "Generated repositories directory")
	authMode := flag.String("auth-mode", environment("PLATFORM_AUTH_MODE", "demo"), "Authentication mode: demo or token")
	tokenRole := flag.String("token-role", environment("PLATFORM_AUTH_ROLE", string(domain.RolePlatformAdmin)), "RBAC role assigned in token mode")
	healthcheck := flag.Bool("healthcheck", false, "Check the local health endpoint")
	showVersion := flag.Bool("version", false, "Print version information")
	flag.Parse()
	if *showVersion {
		fmt.Printf("platform-api %s (commit %s, built %s)\n", version, commit, buildDate)
		return
	}
	if *healthcheck {
		response, err := http.Get(healthURL(*address))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer response.Body.Close()
		_, _ = io.Copy(io.Discard, response.Body)
		if response.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		return
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if *authMode != "demo" && *authMode != "token" {
		logger.Error("invalid authentication mode", "auth_mode", *authMode)
		os.Exit(2)
	}
	configuredRole := domain.Role(*tokenRole)
	if !domain.ValidRole(configuredRole) {
		logger.Error("invalid token role", "role", *tokenRole)
		os.Exit(2)
	}
	token := os.Getenv("PLATFORM_API_TOKEN")
	if *authMode == "token" && len(token) < 16 {
		logger.Error("PLATFORM_API_TOKEN must contain at least 16 characters in token mode")
		os.Exit(2)
	}
	state, err := store.Open(*dataPath)
	if err != nil {
		logger.Error("open state", "error", err)
		os.Exit(1)
	}
	processor := operations.NewProcessor(state, *generatedRoot, os.Getenv("PLATFORM_FAIL_AT_STEP"))
	processor.Start()
	defer processor.Stop()
	server := &http.Server{
		Addr: *address,
		Handler: api.NewServerWithConfig(state, processor, logger, api.Config{
			AuthMode:      *authMode,
			Token:         token,
			TokenRole:     configuredRole,
			Version:       version,
			GeneratedRoot: *generatedRoot,
			UI:            platformweb.Handler(),
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		logger.Info("platform API listening", "address", server.Addr, "data_path", *dataPath, "auth_mode", *authMode, "version", version)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("serve platform API", "error", err)
			os.Exit(1)
		}
	}()
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("shutdown platform API", "error", err)
	}
}

func healthURL(address string) string {
	if strings.HasPrefix(address, ":") {
		address = "127.0.0.1" + address
	}
	if strings.HasPrefix(address, "0.0.0.0:") {
		address = "127.0.0.1:" + strings.TrimPrefix(address, "0.0.0.0:")
	}
	return "http://" + address + "/healthz"
}

func environment(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
