# GitLab MCP Server

A Model Context Protocol (MCP) server for GitLab integration, built with Go using the [mcp-go](https://github.com/mark3labs/mcp-go) framework. This server provides MCP tools for managing and querying GitLab groups and projects through the official GitLab API.

## Features

- **Health Check**: Verify server status and connectivity
- **Group Projects**: List all projects in a group and its subgroups recursively
- **Direct Projects**: List projects directly in a group (excluding subgroups)  
- **Subgroups**: List all subgroups within a parent group
- **Archive Support**: Filter projects by archived status
- **HTTP & Stdio Transport**: Supports both HTTP and stdio transport modes

## Prerequisites

- **Go 1.24+**: Required for building and running the server
- **GitLab Access Token**: Personal access token with appropriate permissions
- **Task Runner** (optional): For using the provided Taskfile commands

## Quick Start

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd gitlab-mcp-server
   ```

2. **Set up environment**
   ```bash
   # Copy environment template
   cp .env.example .env
   
   # Edit .env and add your GitLab token
   export GITLAB_ACCESS_TOKEN=your_gitlab_token_here
   ```

3. **Build and run** (using Task runner)
   ```bash
   # Quick setup and run
   task setup
   task start
   
   # Or manually
   go mod download
   go build -o gitlab-mcp-server ./cmd/gitlab-mcp-server
   ./gitlab-mcp-server
   ```

## Configuration

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `GITLAB_ACCESS_TOKEN` | Yes | GitLab personal access token with API access |

### Transport Modes

- **Stdio** (default): Communicates via stdin/stdout
- **HTTP**: Runs HTTP server on port 8000
  ```bash
  ./gitlab-mcp-server -http
  ```

## Available MCP Tools

### `health_check`
Performs a health check to verify the MCP server is operational.

**Parameters:** None  
**Returns:** Server status and metadata

### `list_all_group_projects`
Lists all projects in a group and its subgroups recursively.

**Parameters:**
- `group_id_or_path` (required): GitLab group ID or path
- `archived` (optional): Filter by archived status (default: false)

**Example:** List all active projects in "mycompany/engineering"

### `list_direct_group_projects`
Lists projects directly in a group (excludes subgroup projects).

**Parameters:**
- `group_id_or_path` (required): GitLab group ID or path

### `list_subgroups`
Lists all direct subgroups within a parent group.

**Parameters:**
- `group_id_or_path` (required): GitLab group ID or path

## Development

### Using Task Runner (Recommended)

```bash
# Show all available commands
task help

# Initial setup
task setup

# Build the project
task build

# Run with environment checks
task start

# Development mode (auto-rebuild)
task dev

# Run tests
task test

# Lint code
task lint

# Clean build artifacts
task clean
```

### Manual Commands

```bash
# Install dependencies
go mod download
go mod tidy

# Build executable
go build -o gitlab-mcp-server ./cmd/gitlab-mcp-server

# Run tests
go test -v ./...

# Format and vet code
go fmt ./...
go vet ./...

# Run the server
./gitlab-mcp-server          # Stdio mode
./gitlab-mcp-server -http    # HTTP mode
```

## Project Structure

```
.
├── .env.example              # Environment template for local setup
├── .github/                  # Automation and workflow configuration
├── AGENTS.md                 # Contributor guide
├── cmd
│   └── gitlab-mcp-server
│       └── main.go           # CLI entry point
├── internal
│   ├── app
│   │   └── server.go         # MCP server wiring and handlers
│   └── gitlab
│       ├── client.go         # GitLab client construction
│       ├── models.go         # Response DTOs for tools
│       └── service.go        # GitLab API integration logic
├── go.mod                    # Go module definition
├── go.sum                    # Go dependency checksums
├── LICENSE                   # MIT License
├── README.md                 # Project documentation
└── Taskfile.yml              # Task runner configuration
```

## Dependencies

- [`github.com/mark3labs/mcp-go`](https://github.com/mark3labs/mcp-go) - MCP server framework
- [`gitlab.com/gitlab-org/api/client-go`](https://gitlab.com/gitlab-org/api/client-go) - Official GitLab API client

## API Integration

The server integrates with GitLab's REST API using the official Go client library. It supports:

- Group and project discovery
- Recursive subgroup traversal  
- Project metadata extraction
- Archive status filtering
- Error handling and logging

## Usage Examples

### With MCP-compatible client:

```json
{
  "method": "tools/call",
  "params": {
    "name": "list_all_group_projects", 
    "arguments": {
      "group_id_or_path": "mycompany/engineering",
      "archived": false
    }
  }
}
```

### Server Response Format:

The server returns detailed project information including:
- Project ID, name, and path
- Repository URLs and web URLs
- Group hierarchy information
- Subgroup project indicators

## Troubleshooting

### Common Issues

1. **Token Authentication**
   ```bash
   # Verify token is set
   task env-check
   ```

2. **Build Issues**
   ```bash
   # Clean and rebuild
   task clean
   task build
   ```

3. **Connection Issues**
   - Verify GitLab token has sufficient permissions
   - Check network connectivity to gitlab.com
   - Ensure group/project paths are correct

### Logging

The server provides detailed logging for:
- Server initialization
- GitLab API connections
- MCP tool registrations
- Request processing
- Error conditions

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Run `task check` to verify code quality
6. Submit a pull request
