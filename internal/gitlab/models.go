package gitlab

import "time"

// Project captures a subset of GitLab project metadata returned to MCP clients.
type Project struct {
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

// Subgroup contains the subset of GitLab subgroup metadata exposed via MCP tools.
type Subgroup struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	FullPath string `json:"full_path"`
	WebURL   string `json:"web_url"`
	ParentID int    `json:"parent_id"`
}

// PipelineSummary captures the key details for pipelines returned to MCP clients.
type PipelineSummary struct {
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
	AgeDays   int        `json:"age_days"`
	AgeYears  float64    `json:"age_years"`
}

// PipelineDeletionError describes a failure encountered when deleting a pipeline.
type PipelineDeletionError struct {
	PipelineID int    `json:"pipeline_id"`
	Error      string `json:"error"`
}

// PipelineDeletionSummary reports the outcome of a bulk pipeline deletion attempt.
type PipelineDeletionSummary struct {
	TotalCandidates int                     `json:"total_candidates"`
	DeletedIDs      []int                   `json:"deleted_ids"`
	Failed          []PipelineDeletionError `json:"failed,omitempty"`
}
