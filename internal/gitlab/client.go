package gitlab

import (
	"fmt"
	"strings"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// NewClient constructs a GitLab API client with the provided token and optional base URL.
func NewClient(token string, baseURL string) (*gitlab.Client, error) {
	trimmedToken := strings.TrimSpace(token)
	if trimmedToken == "" {
		return nil, fmt.Errorf("gitlab token cannot be empty")
	}

	opts := []gitlab.ClientOptionFunc{}
	if url := strings.TrimSpace(baseURL); url != "" {
		opts = append(opts, gitlab.WithBaseURL(url))
	}

	client, err := gitlab.NewClient(trimmedToken, opts...)
	if err != nil {
		return nil, fmt.Errorf("create gitlab client: %w", err)
	}

	return client, nil
}
