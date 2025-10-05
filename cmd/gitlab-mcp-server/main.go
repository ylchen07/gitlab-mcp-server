package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/ylchen07/gitlab-mcp-server/internal/app"
	gitlabsvc "github.com/ylchen07/gitlab-mcp-server/internal/gitlab"
)

func main() {
	useHTTP := flag.Bool("http", false, "Expose the MCP server over HTTP instead of stdio")
	httpAddr := flag.String("addr", ":8000", "HTTP listen address when using --http")
	flag.Parse()

	logger := log.New(os.Stdout, "", log.LstdFlags)
	logger.Println("Starting GitLab MCP Server...")

	token := strings.TrimSpace(os.Getenv("GITLAB_ACCESS_TOKEN"))
	if token == "" {
		logger.Fatal("GITLAB_ACCESS_TOKEN environment variable not set")
	}
	logger.Println("GitLab access token detected")

	serverURL := strings.TrimSpace(os.Getenv("GITLAB_SERVER_URL"))
	if serverURL == "" {
		serverURL = "https://gitlab.com"
		logger.Printf("GITLAB_SERVER_URL not set, defaulting to %s", serverURL)
	} else {
		logger.Printf("Using GitLab server: %s", serverURL)
	}

	client, err := gitlabsvc.NewClient(token, serverURL)
	if err != nil {
		logger.Fatalf("Failed to create GitLab client: %v", err)
	}
	logger.Println("GitLab client initialized")

	gitlabService := gitlabsvc.NewService(client, logger)

	srv := app.NewServer(gitlabService, logger)

	for _, tool := range srv.AvailableTools() {
		logger.Printf("Registered MCP tool %s - %s", tool.Name, tool.Description)
	}

	if *useHTTP {
		logger.Printf("Serving MCP over HTTP on %s", *httpAddr)
		if err := srv.RunHTTP(*httpAddr); err != nil {
			logger.Fatalf("HTTP server terminated: %v", err)
		}
		return
	}

	logger.Println("Serving MCP over stdio")
	if err := srv.RunStdio(); err != nil {
		logger.Fatalf("STDIO server terminated: %v", err)
	}
}
