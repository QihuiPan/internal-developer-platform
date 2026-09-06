package main

import "testing"

func TestHealthURLUsesLoopbackForWildcardListeners(t *testing.T) {
	tests := map[string]string{
		":8080":          "http://127.0.0.1:8080/healthz",
		"0.0.0.0:9090":   "http://127.0.0.1:9090/healthz",
		"127.0.0.1:7000": "http://127.0.0.1:7000/healthz",
	}
	for input, expected := range tests {
		if actual := healthURL(input); actual != expected {
			t.Errorf("healthURL(%q) = %q, want %q", input, actual, expected)
		}
	}
}
