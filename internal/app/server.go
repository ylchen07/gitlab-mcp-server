package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ylchen07/gitlab-mcp-server/internal/gitlab"

	"github.com/mark3labs/mcp-go/mcp"
	serverpkg "github.com/mark3labs/mcp-go/server"
)

const (
	serverName    = "GitLab Project Manager"
	serverVersion = "1.0.0"
)

// ToolInfo describes an MCP tool that has been registered with the server.
type ToolInfo struct {
	Name        string
	Description string
}

// Server coordinates MCP tool registration and request handling for the GitLab integration.
type Server struct {
	mcpServer *serverpkg.MCPServer
	gitlab    *gitlab.Service
	logger    *log.Logger
	tools     []ToolInfo
}

// NewServer constructs a Server backed by the provided GitLab service and logger.
func NewServer(service *gitlab.Service, logger *log.Logger) *Server {
	if logger == nil {
		logger = log.Default()
	}

	s := &Server{
		mcpServer: serverpkg.NewMCPServer(serverName, serverVersion, serverpkg.WithToolCapabilities(false)),
		gitlab:    service,
		logger:    logger,
	}

	s.registerTools()

	return s
}

// AvailableTools returns metadata for each registered MCP tool.
func (s *Server) AvailableTools() []ToolInfo {
	return append([]ToolInfo(nil), s.tools...)
}

// RunStdio starts the server using stdio transport.
func (s *Server) RunStdio() error {
	return serverpkg.ServeStdio(s.mcpServer)
}

// RunHTTP starts the server using HTTP transport on the provided address.
func (s *Server) RunHTTP(addr string) error {
	return serverpkg.NewStreamableHTTPServer(s.mcpServer).Start(addr)
}

func (s *Server) registerTools() {
	s.addTool(mcp.NewTool(
		"health_check",
		mcp.WithDescription("Simple health check to verify the MCP server is working"),
	), s.handleHealthCheck)

	s.addTool(mcp.NewTool(
		"list_all_group_projects",
		mcp.WithDescription("List all projects in a group and its subgroups recursively"),
		mcp.WithString("group_id_or_path", mcp.Required(),
			mcp.Description("GitLab group ID or path"),
		),
		mcp.WithBoolean("archived",
			mcp.Description("Filter by archived status (default: false)"),
		),
	), s.handleListAllGroupProjects)

	s.addTool(mcp.NewTool(
		"list_direct_group_projects",
		mcp.WithDescription("List all projects directly in a group (not including subgroups)"),
		mcp.WithString("group_id_or_path", mcp.Required(),
			mcp.Description("GitLab group ID or path"),
		),
	), s.handleListDirectGroupProjects)

	s.addTool(mcp.NewTool(
		"list_subgroups",
		mcp.WithDescription("List all subgroups in a group"),
		mcp.WithString("group_id_or_path", mcp.Required(),
			mcp.Description("GitLab group ID or path"),
		),
	), s.handleListSubgroups)

	s.addTool(mcp.NewTool(
		"archive_project",
		mcp.WithDescription("Archive a GitLab project (requires Owner role or admin permissions)"),
		mcp.WithString("project_id_or_path", mcp.Required(),
			mcp.Description("GitLab project ID or path with namespace"),
		),
	), s.handleArchiveProject)

	s.addTool(mcp.NewTool(
		"get_project_status",
		mcp.WithDescription("Get detailed status and metadata for a single GitLab project"),
		mcp.WithString("project_id_or_path", mcp.Required(),
			mcp.Description("GitLab project ID or path with namespace"),
		),
	), s.handleGetProjectStatus)

	s.addTool(mcp.NewTool(
		"list_old_pipelines",
		mcp.WithDescription("List all pipelines in a project older than the provided age threshold"),
		mcp.WithString("project_id_or_path", mcp.Required(),
			mcp.Description("GitLab project ID or path with namespace"),
		),
		mcp.WithNumber("older_than_years", mcp.Required(),
			mcp.Description("Age threshold in years; pipelines created before this many years ago will be included"),
		),
	), s.handleListOldPipelines)

	s.addTool(mcp.NewTool(
		"delete_old_pipelines",
		mcp.WithDescription("Delete all pipelines in a project older than the provided age threshold"),
		mcp.WithString("project_id_or_path", mcp.Required(),
			mcp.Description("GitLab project ID or path with namespace"),
		),
		mcp.WithNumber("older_than_years", mcp.Required(),
			mcp.Description("Age threshold in years; pipelines created before this many years ago will be deleted"),
		),
		mcp.WithBoolean("confirm",
			mcp.Description("Set to true to actually delete pipelines; defaults to false for safety"),
		),
	), s.handleDeleteOldPipelines)
}

func (s *Server) addTool(tool mcp.Tool, handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	s.mcpServer.AddTool(tool, handler)
	s.tools = append(s.tools, ToolInfo{Name: tool.Name, Description: tool.Description})
}

func (s *Server) handleHealthCheck(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	result := map[string]any{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"server":    serverName,
		"version":   serverVersion,
	}

	return mcp.NewToolResultText(fmt.Sprintf("Health check successful: %+v", result)), nil
}

func (s *Server) handleListAllGroupProjects(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	groupIDOrPath, err := request.RequireString("group_id_or_path")
	if err != nil {
		return nil, fmt.Errorf("group_id_or_path is required: %w", err)
	}

	archived := request.GetBool("archived", false)

	projects, err := s.gitlab.ListGroupProjectsAll(ctx, groupIDOrPath, archived)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error fetching projects: %v", err)), nil
	}

	jsonData, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error serializing projects: %v", err)), nil
	}

	statusText := "all"
	if archived {
		statusText = "archived"
	}

	return mcp.NewToolResultText(fmt.Sprintf(
		"Found %d %s projects in group %s and its subgroups:\n\n%s",
		len(projects), statusText, groupIDOrPath, string(jsonData),
	)), nil
}

func (s *Server) handleListDirectGroupProjects(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	groupIDOrPath, err := request.RequireString("group_id_or_path")
	if err != nil {
		return nil, fmt.Errorf("group_id_or_path is required: %w", err)
	}

	projects, err := s.gitlab.ListGroupProjects(ctx, groupIDOrPath)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error fetching direct projects: %v", err)), nil
	}

	jsonData, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error serializing projects: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(
		"Found %d direct projects in group %s:\n\n%s",
		len(projects), groupIDOrPath, string(jsonData),
	)), nil
}

func (s *Server) handleListSubgroups(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	groupIDOrPath, err := request.RequireString("group_id_or_path")
	if err != nil {
		return nil, fmt.Errorf("group_id_or_path is required: %w", err)
	}

	subgroups, err := s.gitlab.ListGroupSubgroups(ctx, groupIDOrPath)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error fetching subgroups: %v", err)), nil
	}

	jsonData, err := json.MarshalIndent(subgroups, "", "  ")
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error serializing subgroups: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(
		"Found %d subgroups in group %s:\n\n%s",
		len(subgroups), groupIDOrPath, string(jsonData),
	)), nil
}

func (s *Server) handleArchiveProject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectIDOrPath, err := request.RequireString("project_id_or_path")
	if err != nil {
		return nil, fmt.Errorf("project_id_or_path is required: %w", err)
	}

	project, err := s.gitlab.ArchiveProject(ctx, projectIDOrPath)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error archiving project: %v", err)), nil
	}

	result := map[string]any{
		"success":            true,
		"project_id":         project.ID,
		"project_name":       project.Name,
		"project_path":       project.PathWithNamespace,
		"archived":           project.Archived,
		"web_url":            project.WebURL,
		"archived_timestamp": time.Now().Format(time.RFC3339),
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Project archived but failed to serialize response: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(
		"Project '%s' archived successfully:\n\n%s",
		project.PathWithNamespace, string(jsonData),
	)), nil
}

func (s *Server) handleGetProjectStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectIDOrPath, err := request.RequireString("project_id_or_path")
	if err != nil {
		return nil, fmt.Errorf("project_id_or_path is required: %w", err)
	}

	project, err := s.gitlab.GetProject(ctx, projectIDOrPath)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error fetching project: %v", err)), nil
	}

	result := map[string]any{
		"id":                  project.ID,
		"name":                project.Name,
		"path":                project.Path,
		"path_with_namespace": project.PathWithNamespace,
		"description":         project.Description,
		"web_url":             project.WebURL,
		"clone_url_http":      project.HTTPURLToRepo,
		"clone_url_ssh":       project.SSHURLToRepo,
		"visibility":          project.Visibility,
		"archived":            project.Archived,
		"created_at":          project.CreatedAt,
		"last_activity_at":    project.LastActivityAt,
		"default_branch":      project.DefaultBranch,
		"forks_count":         project.ForksCount,
		"star_count":          project.StarCount,
		"open_issues_count":   project.OpenIssuesCount,
		"topics":              project.Topics,
		"readme_url":          project.ReadmeURL,
	}

	if project.Namespace != nil {
		result["namespace"] = map[string]any{
			"id":        project.Namespace.ID,
			"name":      project.Namespace.Name,
			"path":      project.Namespace.Path,
			"full_path": project.Namespace.FullPath,
			"kind":      project.Namespace.Kind,
		}
	}

	if project.Statistics != nil {
		result["size"] = project.Statistics.RepositorySize
		result["commit_count"] = project.Statistics.CommitCount
		result["storage_size"] = project.Statistics.StorageSize
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error serializing project status: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(
		"Project status for '%s':\n\n%s",
		project.PathWithNamespace, string(jsonData),
	)), nil
}

func (s *Server) handleListOldPipelines(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectIDOrPath, err := request.RequireString("project_id_or_path")
	if err != nil {
		return nil, fmt.Errorf("project_id_or_path is required: %w", err)
	}

	projectIDOrPath = strings.TrimSpace(projectIDOrPath)
	if projectIDOrPath == "" {
		return mcp.NewToolResultText("project_id_or_path cannot be empty"), nil
	}

	years, err := request.RequireInt("older_than_years")
	if err != nil {
		return nil, fmt.Errorf("older_than_years is required: %w", err)
	}

	if years <= 0 {
		return mcp.NewToolResultText("older_than_years must be greater than zero"), nil
	}

	cutoff := time.Now().UTC().AddDate(-years, 0, 0)

	pipelines, err := s.gitlab.ListOldPipelines(ctx, projectIDOrPath, cutoff)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error listing old pipelines: %v", err)), nil
	}

	if len(pipelines) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf(
			"No pipelines in project %s are older than %d years (cutoff %s).",
			projectIDOrPath, years, cutoff.Format(time.RFC3339),
		)), nil
	}

	jsonData, err := json.MarshalIndent(pipelines, "", "  ")
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error serializing pipeline list: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(
		"Found %d pipelines in project %s created before %s (older than %d years):\n\n%s",
		len(pipelines), projectIDOrPath, cutoff.Format(time.RFC3339), years, string(jsonData),
	)), nil
}

func (s *Server) handleDeleteOldPipelines(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectIDOrPath, err := request.RequireString("project_id_or_path")
	if err != nil {
		return nil, fmt.Errorf("project_id_or_path is required: %w", err)
	}

	projectIDOrPath = strings.TrimSpace(projectIDOrPath)
	if projectIDOrPath == "" {
		return mcp.NewToolResultText("project_id_or_path cannot be empty"), nil
	}

	years, err := request.RequireInt("older_than_years")
	if err != nil {
		return nil, fmt.Errorf("older_than_years is required: %w", err)
	}

	if years <= 0 {
		return mcp.NewToolResultText("older_than_years must be greater than zero"), nil
	}

	if !request.GetBool("confirm", false) {
		return mcp.NewToolResultText(
			"Deletion not performed: set confirm=true to delete pipelines after reviewing list_old_pipelines output.",
		), nil
	}

	cutoff := time.Now().UTC().AddDate(-years, 0, 0)

	summary, err := s.gitlab.DeleteOldPipelines(ctx, projectIDOrPath, cutoff)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error deleting old pipelines: %v", err)), nil
	}

	if summary.TotalCandidates == 0 {
		return mcp.NewToolResultText(fmt.Sprintf(
			"No pipelines in project %s are older than %d years (cutoff %s).",
			projectIDOrPath, years, cutoff.Format(time.RFC3339),
		)), nil
	}

	result := map[string]any{
		"project":          projectIDOrPath,
		"cutoff":           cutoff.Format(time.RFC3339),
		"older_than_years": years,
		"total_candidates": summary.TotalCandidates,
		"deleted_count":    len(summary.DeletedIDs),
		"deleted_ids":      summary.DeletedIDs,
	}

	if len(summary.Failed) > 0 {
		result["failed_deletions"] = summary.Failed
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf(
			"Deletion completed but failed to serialize response: %v", err,
		)), nil
	}

	if len(summary.Failed) > 0 {
		return mcp.NewToolResultText(fmt.Sprintf(
			"Deleted %d/%d pipelines older than %d years in project %s (cutoff %s). Some deletions failed:\n\n%s",
			len(summary.DeletedIDs), summary.TotalCandidates, years, projectIDOrPath,
			cutoff.Format(time.RFC3339), string(jsonData),
		)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(
		"Deleted %d/%d pipelines older than %d years in project %s (cutoff %s):\n\n%s",
		len(summary.DeletedIDs), summary.TotalCandidates, years,
		projectIDOrPath, cutoff.Format(time.RFC3339), string(jsonData),
	)), nil
}
