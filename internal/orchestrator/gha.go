package orchestrator

import (
	"context"
	"fmt"

	"github.com/tracker-tv/github-policy-bots/internal/service"
)

type GithubActionsBot struct {
	repos       service.RepositoryService
	policy      service.PolicyService
	remediation service.RemediationService
}

func NewGithubActionsBot(repos service.RepositoryService, policy service.PolicyService, remediation service.RemediationService) *GithubActionsBot {
	return &GithubActionsBot{repos: repos, policy: policy, remediation: remediation}
}

func (b *GithubActionsBot) Run(ctx context.Context) ([]service.RemediationResult, error) {
	repos, err := b.repos.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	var results []service.RemediationResult

	for _, repo := range repos {
		if repo.Archived {
			continue
		}

		repoFiles, err := b.repos.ListFiles(ctx, repo.Name)
		if err != nil {
			fmt.Printf("warning: could not list files for %s: %v\n", repo.Name, err)
			continue
		}

		deviations, err := b.policy.Ensure(ctx, repo, repoFiles)
		if err != nil {
			fmt.Printf("warning: could not check policies for %s: %v\n", repo.Name, err)
			continue
		}

		for _, deviation := range deviations {
			result, err := b.remediation.Remediate(ctx, deviation)
			if err != nil {
				fmt.Printf("warning: could not remediate %s in %s: %v\n",
					deviation.Policy.Name, deviation.Repository.Name, err)
				results = append(results, service.RemediationResult{
					Drift: deviation,
					Error: err,
				})
				continue
			}
			results = append(results, *result)
		}
	}

	return results, nil
}
