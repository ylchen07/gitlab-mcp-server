# GitLab MCP Server

A Model Context Protocol (MCP) server for GitLab integration, built with Go using the [mcp-go](https://mcp-go.dev/) framework.

## Features

- **Health Check**: Verify server status
- **Group Projects**: List all projects in a group and its subgroups recursively
- **Direct Projects**: List projects directly in a group (excluding subgroups)
- **Subgroups**: List all subgroups in a group

## Prerequisites

- Go 1.24 or higher
- GitLab personal access token

## Installation

1. Clone the repository
2. Install dependencies:

   ```bash
   go mod download
   ```

3. Build the server:

   ```bash
   go build -o gitlab-mcp-server main.go
   ```

## Configuration

Set the required environment variable:

```bash
export GITLAB_ACCESS_TOKEN=your_gitlab_token_here
```

Or create a `.env` file based on `.env.example`.

## Usage

Run the MCP server:

```bash
./gitlab-mcp-server
```

The server runs on HTTP transport at `http://localhost:8000` and can be integrated with any MCP-compatible client.

## Available Tools

### `health_check`

Simple health check to verify the MCP server is working.

### `list_all_group_projects`

List all projects in a group and its subgroups recursively.

- **Parameter**: `group_id_or_path` (string) - GitLab group ID or path

### `list_direct_group_projects`

List all projects directly in a group (not including subgroups).

- **Parameter**: `group_id_or_path` (string) - GitLab group ID or path

### `list_subgroups`

List all subgroups in a group.

- **Parameter**: `group_id_or_path` (string) - GitLab group ID or path

## Development

```bash
# Install dependencies
go mod download

# Build
go build -o gitlab-mcp-server main.go

# Run
./gitlab-mcp-server
```

## License

MIT License
