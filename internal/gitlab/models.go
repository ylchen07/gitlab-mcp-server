package gitlab

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
