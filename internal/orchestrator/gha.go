package orchestrator

import (
	"context"
	"fmt"

	"github.com/tracker-tv/github-policy-bots/internal/service"
	"github.com/tracker-tv/github-policy-bots/models"
)

type GithubActionsBot struct {
	repos  service.RepositoryService
	policy service.PolicyService
}

func NewGithubActionsBot(repos service.RepositoryService, policy service.PolicyService) *GithubActionsBot {
	return &GithubActionsBot{repos: repos, policy: policy}
}

func (b *GithubActionsBot) Run(ctx context.Context) ([]models.PolicyViolation, error) {
	repos, err := b.repos.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	var allViolations []models.PolicyViolation

	for _, repo := range repos {
		if repo.Archived {
			continue
		}

		repoFiles, err := b.repos.ListFiles(ctx, repo.Name)
		if err != nil {
			fmt.Printf("warning: could not list files for %s: %v\n", repo.Name, err)
			continue
		}

		violations, err := b.policy.Ensure(ctx, repo, repoFiles)
		if err != nil {
			fmt.Printf("warning: could not check policies for %s: %v\n", repo.Name, err)
			continue
		}

		allViolations = append(allViolations, violations...)
	}

	return allViolations, nil
}
