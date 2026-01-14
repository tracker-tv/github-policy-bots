package service

import (
	"context"
	"fmt"

	"github.com/tracker-tv/github-policy-bots/internal/github"
	"github.com/tracker-tv/github-policy-bots/models"
)

type WorkflowService interface {
	List(ctx context.Context, repo string) ([]*models.WorkflowFile, error)
}

type workflowService struct {
	gh github.Client
}

func NewWorkflowService(ghClient github.Client) WorkflowService {
	return &workflowService{gh: ghClient}
}

func (s *workflowService) List(ctx context.Context, repo string) ([]*models.WorkflowFile, error) {
	_, contents, _, err := s.gh.GetContentsRaw(ctx, repo, ".github/workflows")
	if err != nil {
		return nil, err
	}

	var files []*models.WorkflowFile
	for _, c := range contents {
		content, _, _, err := s.gh.GetContentsRaw(ctx, repo, fmt.Sprintf(".github/workflows/%s", c.GetName()))
		if err != nil {
			continue
		}

		decoded, err := content.GetContent()
		if err != nil {
			return nil, err
		}

		files = append(files, &models.WorkflowFile{
			Name:    c.GetName(),
			Path:    c.GetPath(),
			Content: decoded,
		})
	}

	return files, nil
}
