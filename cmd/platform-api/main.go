package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/QihuiPan/internal-developer-platform/internal/api"
	"github.com/QihuiPan/internal-developer-platform/internal/operations"
	"github.com/QihuiPan/internal-developer-platform/internal/store"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--healthcheck" {
		response, err := http.Get("http://127.0.0.1:8080/healthz")
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
	dataPath := environment("PLATFORM_DATA_PATH", filepath.Join(".platform", "state.json"))
	generatedRoot := environment("GENERATED_SERVICES_DIR", filepath.Join(".platform", "generated"))
	state, err := store.Open(dataPath)
	if err != nil {
		logger.Error("open state", "error", err)
		os.Exit(1)
	}
	processor := operations.NewProcessor(state, generatedRoot, os.Getenv("PLATFORM_FAIL_AT_STEP"))
	processor.Start()
	defer processor.Stop()
	server := &http.Server{
		Addr:              environment("PLATFORM_ADDRESS", ":8080"),
		Handler:           api.NewServer(state, processor, logger),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		logger.Info("platform API listening", "address", server.Addr, "data_path", dataPath)
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

func environment(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
