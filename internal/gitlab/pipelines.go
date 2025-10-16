package gitlab

import (
	"context"
	"fmt"
	"math"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

const pipelinePageSize = 100

// ListOldPipelines returns pipelines for the given project created before the specified timestamp.
func (s *Service) ListOldPipelines(ctx context.Context, projectIDOrPath string, before time.Time) ([]PipelineSummary, error) {
	cutoff := before.UTC()

	opts := &gitlab.ListProjectPipelinesOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: pipelinePageSize,
			Page:    1,
		},
		CreatedBefore: gitlab.Ptr(cutoff),
		OrderBy:       gitlab.Ptr("created_at"),
		Sort:          gitlab.Ptr("asc"),
	}

	var results []PipelineSummary

	for {
		pipelines, resp, err := s.client.Pipelines.ListProjectPipelines(projectIDOrPath, opts, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("list project pipelines: %w", err)
		}

		for _, pipeline := range pipelines {
			if pipeline == nil {
				continue
			}

			var createdAtPtr *time.Time
			var updatedAtPtr *time.Time

			if pipeline.CreatedAt != nil {
				created := pipeline.CreatedAt.UTC()
				if !created.Before(cutoff) {
					continue
				}
				createdAtPtr = gitlab.Ptr(created)
			}

			if pipeline.UpdatedAt != nil {
				updated := pipeline.UpdatedAt.UTC()
				updatedAtPtr = gitlab.Ptr(updated)
			}

			ageDays, ageYears := pipelineAge(createdAtPtr)

			results = append(results, PipelineSummary{
				ID:        pipeline.ID,
				IID:       pipeline.IID,
				ProjectID: pipeline.ProjectID,
				Status:    pipeline.Status,
				Source:    pipeline.Source,
				Ref:       pipeline.Ref,
				SHA:       pipeline.SHA,
				WebURL:    pipeline.WebURL,
				CreatedAt: createdAtPtr,
				UpdatedAt: updatedAtPtr,
				AgeDays:   ageDays,
				AgeYears:  ageYears,
			})
		}

		if resp == nil || resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	return results, nil
}

// DeleteOldPipelines deletes all pipelines for the given project created before the specified timestamp.
func (s *Service) DeleteOldPipelines(ctx context.Context, projectIDOrPath string, before time.Time) (*PipelineDeletionSummary, error) {
	pipelines, err := s.ListOldPipelines(ctx, projectIDOrPath, before)
	if err != nil {
		return nil, err
	}

	result := &PipelineDeletionSummary{
		TotalCandidates: len(pipelines),
	}

	if len(pipelines) == 0 {
		return result, nil
	}

	for _, pipeline := range pipelines {
		if _, err := s.client.Pipelines.DeletePipeline(projectIDOrPath, pipeline.ID, gitlab.WithContext(ctx)); err != nil {
			s.log.Printf("error deleting pipeline %d in project %s: %v", pipeline.ID, projectIDOrPath, err)
			result.Failed = append(result.Failed, PipelineDeletionError{
				PipelineID: pipeline.ID,
				Error:      err.Error(),
			})
			continue
		}

		result.DeletedIDs = append(result.DeletedIDs, pipeline.ID)
	}

	return result, nil
}

func pipelineAge(createdAt *time.Time) (int, float64) {
	if createdAt == nil {
		return -1, -1
	}

	duration := time.Since(*createdAt)
	if duration < 0 {
		return -1, -1
	}

	ageDays := int(duration.Hours() / 24)
	ageYears := math.Round((duration.Hours()/24/365.25)*100) / 100

	return ageDays, ageYears
}
