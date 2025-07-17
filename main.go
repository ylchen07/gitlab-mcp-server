package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// GitLabProject represents a GitLab project with its metadata
type GitLabProject struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	Path              string `json:"path"`
	PathWithNamespace string `json:"path_with_namespace"`
	WebURL            string `json:"web_url"`
	CloneURL          string `json:"clone_url"`
	GroupPath         string `json:"group_path"`
	IsSubgroupProject bool   `json:"is_subgroup_project"`
	SubgroupFullPath  string `json:"subgroup_full_path,omitempty"`
}

// GitLabSubgroup represents a GitLab subgroup with its metadata
type GitLabSubgroup struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	FullPath string `json:"full_path"`
	WebURL   string `json:"web_url"`
	ParentID int    `json:"parent_id"`
}

// GitLabMCPServer holds the MCP server instance and GitLab client
type GitLabMCPServer struct {
	mcpServer *server.MCPServer
	client    *gitlab.Client
}

// NewGitLabMCPServer creates a new GitLab MCP server instance
func NewGitLabMCPServer() (*GitLabMCPServer, error) {
	// Get GitLab token from environment
	log.Println("Checking for GitLab access token...")
	token := os.Getenv("GITLAB_ACCESS_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITLAB_ACCESS_TOKEN environment variable not set")
	}
	log.Println("GitLab access token found")

	// Create GitLab client
	log.Println("Creating GitLab client...")
	gitlabClient, err := gitlab.NewClient(token, gitlab.WithBaseURL("https://gitlab.com"))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}
	log.Println("GitLab client created successfully")

	// Create MCP server
	log.Println("Creating MCP server...")
	mcpServer := server.NewMCPServer(
		"GitLab Project Manager",
		"1.0.0",
		server.WithToolCapabilities(false),
	)
	log.Println("MCP server created successfully")

	gitlabServer := &GitLabMCPServer{
		mcpServer: mcpServer,
		client:    gitlabClient,
	}

	// Register tools
	log.Println("Registering MCP tools...")
	gitlabServer.registerTools()
	log.Println("MCP tools registered successfully")

	return gitlabServer, nil
}

// registerTools registers all MCP tools
func (s *GitLabMCPServer) registerTools() {
	// Health check tool
	healthCheckTool := mcp.NewTool("health_check",
		mcp.WithDescription("Simple health check to verify the MCP server is working"),
	)
	s.mcpServer.AddTool(healthCheckTool, s.handleHealthCheck)

	// List all group projects tool
	listAllProjectsTool := mcp.NewTool("list_all_group_projects",
		mcp.WithDescription("List all projects in a group and its subgroups recursively"),
		mcp.WithString("group_id_or_path", mcp.Required(),
			mcp.Description("GitLab group ID or path"),
		),
		mcp.WithBoolean("archived",
			mcp.Description("Filter by archived status (default: false)"),
		),
	)
	s.mcpServer.AddTool(listAllProjectsTool, s.handleListAllGroupProjects)

	// List direct group projects tool
	listDirectProjectsTool := mcp.NewTool("list_direct_group_projects",
		mcp.WithDescription("List all projects directly in a group (not including subgroups)"),
		mcp.WithString("group_id_or_path", mcp.Required(),
			mcp.Description("GitLab group ID or path"),
		),
	)
	s.mcpServer.AddTool(listDirectProjectsTool, s.handleListDirectGroupProjects)

	// List subgroups tool
	listSubgroupsTool := mcp.NewTool("list_subgroups",
		mcp.WithDescription("List all subgroups in a group"),
		mcp.WithString("group_id_or_path", mcp.Required(),
			mcp.Description("GitLab group ID or path"),
		),
	)
	s.mcpServer.AddTool(listSubgroupsTool, s.handleListSubgroups)
}

// handleHealthCheck handles the health check tool
func (s *GitLabMCPServer) handleHealthCheck(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	result := map[string]any{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"server":    "GitLab Project Manager",
		"version":   "1.0.0",
	}

	return mcp.NewToolResultText(fmt.Sprintf("Health check successful: %+v", result)), nil
}

// handleListAllGroupProjects handles listing all projects in a group and its subgroups
func (s *GitLabMCPServer) handleListAllGroupProjects(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract group_id_or_path from arguments
	groupIDOrPath, err := request.RequireString("group_id_or_path")
	if err != nil {
		return nil, fmt.Errorf("group_id_or_path is required: %w", err)
	}

	// Extract archived parameter (optional, defaults to false)
	archived := false
	if archivedParam := request.GetBool("archived", false); archivedParam {
		archived = true
	}

	// Get all projects recursively
	projects, err := s.listGroupProjectsAll(ctx, groupIDOrPath, archived)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error fetching projects: %v", err)), nil
	}

	// Convert projects to JSON
	jsonData, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error serializing projects: %v", err)), nil
	}

	statusText := "all"
	if archived {
		statusText = "archived"
	}

	return mcp.NewToolResultText(fmt.Sprintf("Found %d %s projects in group %s and its subgroups:\n\n%s", len(projects), statusText, groupIDOrPath, string(jsonData))), nil
}

// handleListDirectGroupProjects handles listing projects directly in a group
func (s *GitLabMCPServer) handleListDirectGroupProjects(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract group_id_or_path from arguments
	groupIDOrPath, err := request.RequireString("group_id_or_path")
	if err != nil {
		return nil, fmt.Errorf("group_id_or_path is required: %w", err)
	}

	// Get direct projects only
	projects, err := s.listGroupProjects(ctx, groupIDOrPath)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error fetching direct projects: %v", err)), nil
	}

	// Convert projects to JSON
	jsonData, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error serializing projects: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Found %d direct projects in group %s:\n\n%s", len(projects), groupIDOrPath, string(jsonData))), nil
}

// handleListSubgroups handles listing subgroups in a group
func (s *GitLabMCPServer) handleListSubgroups(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract group_id_or_path from arguments
	groupIDOrPath, err := request.RequireString("group_id_or_path")
	if err != nil {
		return nil, fmt.Errorf("group_id_or_path is required: %w", err)
	}

	// Get subgroups
	subgroups, err := s.listGroupSubgroups(ctx, groupIDOrPath)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error fetching subgroups: %v", err)), nil
	}

	// Convert subgroups to JSON
	jsonData, err := json.MarshalIndent(subgroups, "", "  ")
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error serializing subgroups: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Found %d subgroups in group %s:\n\n%s", len(subgroups), groupIDOrPath, string(jsonData))), nil
}

// listGroupProjectsAll lists all projects in a group and its subgroups recursively
func (s *GitLabMCPServer) listGroupProjectsAll(ctx context.Context, groupIDOrPath string, archived bool) ([]GitLabProject, error) {
	var allProjects []GitLabProject

	// Get the main group
	group, _, err := s.client.Groups.GetGroup(groupIDOrPath, nil, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	// Prepare options with archived filter
	options := &gitlab.ListGroupProjectsOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
	}
	if archived {
		options.Archived = &archived
	}

	// Get projects directly in this group
	directProjects, _, err := s.client.Groups.ListGroupProjects(group.ID, options, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to list group projects: %w", err)
	}

	// Convert direct projects
	for _, project := range directProjects {
		allProjects = append(allProjects, GitLabProject{
			ID:                project.ID,
			Name:              project.Name,
			Path:              project.Path,
			PathWithNamespace: project.PathWithNamespace,
			WebURL:            project.WebURL,
			CloneURL:          project.HTTPURLToRepo,
			GroupPath:         group.Path,
			IsSubgroupProject: false,
		})
	}

	// Get all descendant groups (subgroups)
	descendantGroups, _, err := s.client.Groups.ListDescendantGroups(group.ID, &gitlab.ListDescendantGroupsOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
	}, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to list descendant groups: %w", err)
	}

	// Get projects from each subgroup
	for _, subgroup := range descendantGroups {
		subgroupProjects, _, err := s.client.Groups.ListGroupProjects(subgroup.ID, options, gitlab.WithContext(ctx))
		if err != nil {
			log.Printf("Error getting projects from subgroup %s: %v", subgroup.FullPath, err)
			continue
		}

		// Convert subgroup projects
		for _, project := range subgroupProjects {
			allProjects = append(allProjects, GitLabProject{
				ID:                project.ID,
				Name:              project.Name,
				Path:              project.Path,
				PathWithNamespace: project.PathWithNamespace,
				WebURL:            project.WebURL,
				CloneURL:          project.HTTPURLToRepo,
				GroupPath:         subgroup.Path,
				IsSubgroupProject: true,
				SubgroupFullPath:  subgroup.FullPath,
			})
		}
	}

	return allProjects, nil
}

// listGroupProjects lists projects directly in a group
func (s *GitLabMCPServer) listGroupProjects(ctx context.Context, groupIDOrPath string) ([]GitLabProject, error) {
	// Get the group
	group, _, err := s.client.Groups.GetGroup(groupIDOrPath, nil, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	// Get projects directly in this group only
	directProjects, _, err := s.client.Groups.ListGroupProjects(group.ID, &gitlab.ListGroupProjectsOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
	}, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to list group projects: %w", err)
	}

	// Convert projects
	var projects []GitLabProject
	for _, project := range directProjects {
		projects = append(projects, GitLabProject{
			ID:                project.ID,
			Name:              project.Name,
			Path:              project.Path,
			PathWithNamespace: project.PathWithNamespace,
			WebURL:            project.WebURL,
			CloneURL:          project.HTTPURLToRepo,
			GroupPath:         group.Path,
			IsSubgroupProject: false,
		})
	}

	return projects, nil
}

// listGroupSubgroups lists subgroups in a group
func (s *GitLabMCPServer) listGroupSubgroups(ctx context.Context, groupIDOrPath string) ([]GitLabSubgroup, error) {
	// Get the group
	group, _, err := s.client.Groups.GetGroup(groupIDOrPath, nil, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	// Get direct subgroups
	directSubgroups, _, err := s.client.Groups.ListSubGroups(group.ID, &gitlab.ListSubGroupsOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
	}, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to list subgroups: %w", err)
	}

	// Convert subgroups
	var subgroups []GitLabSubgroup
	for _, subgroup := range directSubgroups {
		subgroups = append(subgroups, GitLabSubgroup{
			ID:       subgroup.ID,
			Name:     subgroup.Name,
			Path:     subgroup.Path,
			FullPath: subgroup.FullPath,
			WebURL:   subgroup.WebURL,
			ParentID: group.ID,
		})
	}

	return subgroups, nil
}

// Run starts the MCP server with the specified transport mode
func (s *GitLabMCPServer) Run(useHTTP bool) error {
	if useHTTP {
		log.Println("MCP Server is now running on http://localhost:8000")
		log.Println("Ready to handle MCP requests via HTTP...")
		return server.NewStreamableHTTPServer(s.mcpServer).Start(":8000")
	} else {
		log.Println("MCP Server starting with stdio transport")
		log.Println("Ready to handle MCP requests via stdin/stdout...")
		return server.ServeStdio(s.mcpServer)
	}
}

func main() {
	// Parse command line flags
	useHTTP := flag.Bool("http", false, "Use HTTP transport instead of stdio")
	flag.Parse()

	log.Println("Starting GitLab MCP Server...")

	// Create GitLab MCP server
	log.Println("Initializing GitLab MCP server...")
	server, err := NewGitLabMCPServer()
	if err != nil {
		log.Fatal("Failed to create GitLab MCP server:", err)
	}

	log.Println("Server initialized successfully")
	log.Println("Available MCP tools:")
	log.Println("  - health_check: Simple health check")
	log.Println("  - list_all_group_projects: List all projects in a group and subgroups (archived=true for archived only)")
	log.Println("  - list_direct_group_projects: List projects directly in a group")
	log.Println("  - list_subgroups: List subgroups in a group")

	// Run the server
	if err := server.Run(*useHTTP); err != nil {
		log.Fatal("Failed to run GitLab MCP server:", err)
	}
}
