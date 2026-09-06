package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientSendsIdentityAndBearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Actor") != "alice" || r.Header.Get("X-Role") != "developer" {
			t.Error("identity headers were not sent")
		}
		if r.Header.Get("Authorization") != "Bearer secret-token" {
			t.Error("Bearer token was not sent")
		}
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	}))
	defer server.Close()
	c := &client{address: server.URL, actor: "alice", role: "developer", token: "secret-token", http: &http.Client{Timeout: time.Second}}
	if _, err := c.request(http.MethodGet, "/test", nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestClientReturnsStructuredAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, `{"code":"SERVICE_EXISTS","message":"already exists"}`)
	}))
	defer server.Close()
	c := &client{address: server.URL, actor: "alice", role: "developer", http: &http.Client{Timeout: time.Second}}
	_, err := c.request(http.MethodGet, "/test", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "SERVICE_EXISTS") {
		t.Fatalf("unexpected error: %v", err)
	}
}
