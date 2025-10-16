package app

import (
	"context"
	"io"
	"log"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ylchen07/gitlab-mcp-server/internal/gitlab"
)

func TestNewServerRegistersAllTools(t *testing.T) {
	service := gitlab.NewService(nil, log.New(io.Discard, "", 0))
	server := NewServer(service, log.New(io.Discard, "", 0))

	tools := server.AvailableTools()
	if len(tools) < 7 {
		t.Fatalf("expected multiple tools to be registered, got %d", len(tools))
	}

	expected := map[string]bool{
		"health_check":               true,
		"list_all_group_projects":    true,
		"list_direct_group_projects": true,
		"list_subgroups":             true,
		"archive_project":            true,
		"get_project_status":         true,
		"list_old_pipelines":         true,
		"delete_old_pipelines":       true,
	}

	for _, tool := range tools {
		delete(expected, tool.Name)
	}

	if len(expected) != 0 {
		t.Fatalf("missing tool registrations: %v", expected)
	}
}

func TestHandleHealthCheck(t *testing.T) {
	server := NewServer(gitlab.NewService(nil, log.New(io.Discard, "", 0)), log.New(io.Discard, "", 0))

	result, err := server.handleHealthCheck(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("health check returned error: %v", err)
	}

	var combined strings.Builder
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			combined.WriteString(textContent.Text)
		}
	}

	if !strings.Contains(combined.String(), "healthy") {
		t.Fatalf("expected health check output to mention healthy, got %q", combined.String())
	}
}
