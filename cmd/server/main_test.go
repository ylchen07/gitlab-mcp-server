package main

import (
	"errors"
	"io"
	"log"
	"strings"
	"testing"

	"github.com/ylchen07/gitlab-mcp-server/internal/app"
)

func TestRunMissingToken(t *testing.T) {
	logger := log.New(io.Discard, "", 0)

	err := run([]string{"gitlab-mcp-server"}, func(string) string { return "" }, logger, func(*app.Server, bool, string) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "GITLAB_ACCESS_TOKEN") {
		t.Fatalf("expected missing token error, got %v", err)
	}
}

func TestRunStartsServerWithDefaults(t *testing.T) {
	logger := log.New(io.Discard, "", 0)

	env := map[string]string{
		"GITLAB_ACCESS_TOKEN": "token",
	}

	var (
		called  bool
		useHTTP bool
		addr    string
	)

	err := run([]string{"gitlab-mcp-server"}, func(key string) string { return env[key] }, logger,
		func(srv *app.Server, serveHTTP bool, httpAddr string) error {
			if srv == nil {
				t.Fatal("expected server instance")
			}
			called = true
			useHTTP = serveHTTP
			addr = httpAddr
			return nil
		},
	)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if !called {
		t.Fatal("expected starter to be called")
	}
	if useHTTP {
		t.Fatal("expected stdio mode by default")
	}
	if addr != ":8000" {
		t.Fatalf("expected default addr :8000, got %s", addr)
	}
}

func TestRunStartsServerWithHTTP(t *testing.T) {
	logger := log.New(io.Discard, "", 0)

	env := map[string]string{
		"GITLAB_ACCESS_TOKEN": "token",
		"GITLAB_SERVER_URL":   "https://example.com",
	}

	var (
		called  bool
		useHTTP bool
		addr    string
	)

	err := run([]string{"gitlab-mcp-server", "-http", "-addr", ":9999"}, func(key string) string { return env[key] }, logger,
		func(srv *app.Server, serveHTTP bool, httpAddr string) error {
			called = true
			useHTTP = serveHTTP
			addr = httpAddr
			return errors.New("stop")
		},
	)
	if err == nil || !strings.Contains(err.Error(), "stop") {
		t.Fatalf("expected propagated error, got %v", err)
	}
	if !called {
		t.Fatal("expected starter to be called")
	}
	if !useHTTP {
		t.Fatal("expected HTTP mode")
	}
	if addr != ":9999" {
		t.Fatalf("expected addr :9999, got %s", addr)
	}
}
