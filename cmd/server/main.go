package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/ylchen07/gitlab-mcp-server/internal/app"
	gitlabsvc "github.com/ylchen07/gitlab-mcp-server/internal/gitlab"
)

type serverStarter func(*app.Server, bool, string) error

func run(args []string, getenv func(string) string, logger *log.Logger, start serverStarter) error {
	flagSet := flag.NewFlagSet("gitlab-mcp-server", flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)

	useHTTP := flagSet.Bool("http", false, "Expose the MCP server over HTTP instead of stdio")
	httpAddr := flagSet.String("addr", ":8000", "HTTP listen address when using --http")

	if err := flagSet.Parse(args[1:]); err != nil {
		return err
	}

	logger.Println("Starting GitLab MCP Server...")

	token := strings.TrimSpace(getenv("GITLAB_ACCESS_TOKEN"))
	if token == "" {
		return fmt.Errorf("GITLAB_ACCESS_TOKEN environment variable not set")
	}
	logger.Println("GitLab access token detected")

	serverURL := strings.TrimSpace(getenv("GITLAB_SERVER_URL"))
	if serverURL == "" {
		serverURL = "https://gitlab.com"
		logger.Printf("GITLAB_SERVER_URL not set, defaulting to %s", serverURL)
	} else {
		logger.Printf("Using GitLab server: %s", serverURL)
	}

	client, err := gitlabsvc.NewClient(token, serverURL)
	if err != nil {
		return fmt.Errorf("failed to create GitLab client: %w", err)
	}
	logger.Println("GitLab client initialized")

	gitlabService := gitlabsvc.NewService(client, logger)

	srv := app.NewServer(gitlabService, logger)

	for _, tool := range srv.AvailableTools() {
		logger.Printf("Registered MCP tool %s - %s", tool.Name, tool.Description)
	}

	return start(srv, *useHTTP, *httpAddr)
}

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	start := func(srv *app.Server, useHTTP bool, addr string) error {
		if useHTTP {
			logger.Printf("Serving MCP over HTTP on %s", addr)
			if err := srv.RunHTTP(addr); err != nil {
				return fmt.Errorf("HTTP server terminated: %w", err)
			}
			return nil
		}

		logger.Println("Serving MCP over stdio")
		if err := srv.RunStdio(); err != nil {
			return fmt.Errorf("STDIO server terminated: %w", err)
		}

		return nil
	}

	if err := run(os.Args, os.Getenv, logger, start); err != nil {
		logger.Fatal(err)
	}
}
