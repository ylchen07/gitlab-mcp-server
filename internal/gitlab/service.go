package gitlab

import (
	"context"
	"fmt"
	"log"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// Service wraps a GitLab API client and exposes higher-level operations for MCP tools.
type Service struct {
	client *gitlab.Client
	log    *log.Logger
}

// NewService creates a new Service instance using the provided client and logger.
func NewService(client *gitlab.Client, logger *log.Logger) *Service {
	if logger == nil {
		logger = log.Default()
	}

	return &Service{
		client: client,
		log:    logger,
	}
}

// ListGroupProjectsAll returns all projects within the specified group and any descendant subgroups.
func (s *Service) ListGroupProjectsAll(ctx context.Context, groupIDOrPath string, archived bool) ([]Project, error) {
	group, _, err := s.client.Groups.GetGroup(groupIDOrPath, nil, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("get group: %w", err)
	}

	opts := &gitlab.ListGroupProjectsOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
	}
	if archived {
		opts.Archived = gitlab.Ptr(true)
	}

	directProjects, _, err := s.client.Groups.ListGroupProjects(group.ID, opts, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("list group projects: %w", err)
	}

	var allProjects []Project
	for _, project := range directProjects {
		allProjects = append(allProjects, Project{
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

	descendantGroups, _, err := s.client.Groups.ListDescendantGroups(group.ID, &gitlab.ListDescendantGroupsOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
	}, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("list descendant groups: %w", err)
	}

	for _, subgroup := range descendantGroups {
		subgroupProjects, _, err := s.client.Groups.ListGroupProjects(subgroup.ID, opts, gitlab.WithContext(ctx))
		if err != nil {
			s.log.Printf("error listing projects for subgroup %s: %v", subgroup.FullPath, err)
			continue
		}

		for _, project := range subgroupProjects {
			allProjects = append(allProjects, Project{
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

// ListGroupProjects returns projects that belong directly to the specified group.
func (s *Service) ListGroupProjects(ctx context.Context, groupIDOrPath string) ([]Project, error) {
	group, _, err := s.client.Groups.GetGroup(groupIDOrPath, nil, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("get group: %w", err)
	}

	directProjects, _, err := s.client.Groups.ListGroupProjects(group.ID, &gitlab.ListGroupProjectsOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
	}, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("list group projects: %w", err)
	}

	var projects []Project
	for _, project := range directProjects {
		projects = append(projects, Project{
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

// ListGroupSubgroups returns the subgroups directly under the specified group.
func (s *Service) ListGroupSubgroups(ctx context.Context, groupIDOrPath string) ([]Subgroup, error) {
	group, _, err := s.client.Groups.GetGroup(groupIDOrPath, nil, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("get group: %w", err)
	}

	subgroups, _, err := s.client.Groups.ListSubGroups(group.ID, &gitlab.ListSubGroupsOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
	}, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("list subgroups: %w", err)
	}

	var result []Subgroup
	for _, subgroup := range subgroups {
		result = append(result, Subgroup{
			ID:       subgroup.ID,
			Name:     subgroup.Name,
			Path:     subgroup.Path,
			FullPath: subgroup.FullPath,
			WebURL:   subgroup.WebURL,
			ParentID: group.ID,
		})
	}

	return result, nil
}

// ArchiveProject archives the specified project.
func (s *Service) ArchiveProject(ctx context.Context, projectIDOrPath string) (*gitlab.Project, error) {
	project, _, err := s.client.Projects.ArchiveProject(projectIDOrPath, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("archive project: %w", err)
	}

	return project, nil
}

// GetProject retrieves a project by ID or path.
func (s *Service) GetProject(ctx context.Context, projectIDOrPath string) (*gitlab.Project, error) {
	project, _, err := s.client.Projects.GetProject(projectIDOrPath, nil, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}

	return project, nil
}
