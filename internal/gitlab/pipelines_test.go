package gitlab

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	gitlabclient "gitlab.com/gitlab-org/api/client-go"
)

type pipelineResponse struct {
	ID        int        `json:"id"`
	IID       int        `json:"iid"`
	ProjectID int        `json:"project_id"`
	Status    string     `json:"status"`
	Source    string     `json:"source"`
	Ref       string     `json:"ref"`
	SHA       string     `json:"sha"`
	WebURL    string     `json:"web_url"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

type fakeGitLabServer struct {
	t              *testing.T
	projectPath    string
	pipelines      []pipelineResponse
	deleteFailures map[int]bool

	mu          sync.Mutex
	lastQuery   url.Values
	lastPath    string
	deleteCalls []int
}

func (f *fakeGitLabServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.lastPath = r.URL.Path
	f.mu.Unlock()

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, f.projectPath) && strings.HasSuffix(r.URL.Path, "/pipelines"):
		f.mu.Lock()
		f.lastQuery = r.URL.Query()
		pipelines := append([]pipelineResponse(nil), f.pipelines...)
		f.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(pipelines); err != nil {
			f.t.Fatalf("encodes pipelines: %v", err)
		}
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, f.projectPath) && strings.Contains(r.URL.Path, "/pipelines/"):
		idStr := path.Base(r.URL.Path)
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		f.mu.Lock()
		f.deleteCalls = append(f.deleteCalls, id)
		f.lastPath = r.URL.Path
		fail := f.deleteFailures != nil && f.deleteFailures[id]
		f.mu.Unlock()

		if fail {
			http.Error(w, "delete failed", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	default:
		http.NotFound(w, r)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func setupPipelineService(t *testing.T, project string, pipelines []pipelineResponse, deleteFailures map[int]bool) (*Service, *fakeGitLabServer) {
	t.Helper()

	projectPath := "/api/v4/projects/" + project
	fake := &fakeGitLabServer{
		t:              t,
		projectPath:    projectPath,
		pipelines:      append([]pipelineResponse(nil), pipelines...),
		deleteFailures: deleteFailures,
	}

	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			recorder := httptest.NewRecorder()
			fake.ServeHTTP(recorder, r)
			return recorder.Result(), nil
		}),
	}

	client, err := gitlabclient.NewClient(
		"test-token",
		gitlabclient.WithBaseURL("http://example.com/api/v4"),
		gitlabclient.WithHTTPClient(httpClient),
	)
	if err != nil {
		t.Fatalf("create gitlab client: %v", err)
	}

	service := NewService(client, log.New(io.Discard, "", 0))

	return service, fake
}

func TestListOldPipelines(t *testing.T) {
	project := "group/project"
	oldCreated := time.Now().AddDate(-3, 0, 0).UTC()
	newCreated := time.Now().AddDate(-1, 0, 0).UTC()

	pipelines := []pipelineResponse{
		{
			ID:        101,
			IID:       1,
			ProjectID: 999,
			Status:    "success",
			Source:    "push",
			Ref:       "main",
			SHA:       "abc123",
			WebURL:    "https://example.com",
			CreatedAt: &oldCreated,
			UpdatedAt: &oldCreated,
		},
		{
			ID:        202,
			IID:       2,
			ProjectID: 999,
			Status:    "failed",
			Source:    "web",
			Ref:       "feature",
			SHA:       "def456",
			WebURL:    "https://example.com/newer",
			CreatedAt: &newCreated,
			UpdatedAt: &newCreated,
		},
	}

	service, fake := setupPipelineService(t, project, pipelines, nil)

	cutoff := time.Now().UTC().AddDate(-2, 0, 0)
	result, err := service.ListOldPipelines(context.Background(), project, cutoff)
	if err != nil {
		fake.mu.Lock()
		path := fake.lastPath
		fake.mu.Unlock()
		t.Fatalf("ListOldPipelines returned error: %v (path: %s)", err, path)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(result))
	}

	p := result[0]
	if p.ID != 101 {
		t.Errorf("expected pipeline ID 101, got %d", p.ID)
	}

	if p.AgeDays < 700 {
		t.Errorf("expected AgeDays >= 700, got %d", p.AgeDays)
	}

	if p.AgeYears < 2.0 {
		t.Errorf("expected AgeYears >= 2.0, got %.2f", p.AgeYears)
	}

	fake.mu.Lock()
	createdBefore := fake.lastQuery.Get("created_before")
	fake.mu.Unlock()
	if createdBefore == "" {
		t.Error("expected created_before query parameter to be set")
	}
}

func TestDeleteOldPipelines(t *testing.T) {
	project := "group/project"
	created := time.Now().AddDate(-5, 0, 0).UTC()

	pipelines := []pipelineResponse{
		{
			ID:        321,
			IID:       3,
			ProjectID: 42,
			Status:    "success",
			Source:    "push",
			Ref:       "main",
			SHA:       "deadbeef",
			WebURL:    "https://example.com/old",
			CreatedAt: &created,
			UpdatedAt: &created,
		},
	}

	service, fake := setupPipelineService(t, project, pipelines, nil)

	cutoff := time.Now().UTC().AddDate(-2, 0, 0)
	summary, err := service.DeleteOldPipelines(context.Background(), project, cutoff)
	if err != nil {
		fake.mu.Lock()
		path := fake.lastPath
		fake.mu.Unlock()
		t.Fatalf("DeleteOldPipelines returned error: %v (path: %s)", err, path)
	}

	if summary.TotalCandidates != 1 {
		t.Errorf("expected TotalCandidates 1, got %d", summary.TotalCandidates)
	}

	if len(summary.DeletedIDs) != 1 || summary.DeletedIDs[0] != 321 {
		t.Fatalf("unexpected deleted IDs: %#v", summary.DeletedIDs)
	}

	if len(summary.Failed) != 0 {
		t.Fatalf("expected no failed deletions, got %#v", summary.Failed)
	}

	fake.mu.Lock()
	deleteCalls := append([]int(nil), fake.deleteCalls...)
	fake.mu.Unlock()

	if len(deleteCalls) != 1 || deleteCalls[0] != 321 {
		t.Fatalf("expected delete call for pipeline 321, got %v", deleteCalls)
	}
}

func TestPipelineAge(t *testing.T) {
	if days, years := pipelineAge(nil); days != -1 || years != -1 {
		t.Errorf("expected (-1, -1) for nil input, got (%d, %.2f)", days, years)
	}

	future := time.Now().Add(24 * time.Hour)
	if days, years := pipelineAge(&future); days != -1 || years != -1 {
		t.Errorf("expected (-1, -1) for future input, got (%d, %.2f)", days, years)
	}

	past := time.Now().AddDate(0, 0, -10)
	days, years := pipelineAge(&past)
	if days < 10 {
		t.Errorf("expected at least 10 days, got %d", days)
	}
	if math.Abs(years-10.0/365.25) > 0.01 {
		t.Errorf("unexpected years value: %.4f", years)
	}
}
