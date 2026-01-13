package orchestrator

import (
	"context"
	"fmt"

	"github.com/tracker-tv/github-policy-bots/internal/service"
)

type GithubActionsBot struct {
	repos     service.RepositoryService
	workflows service.WorkflowService
}

func NewGithubActionsBot(repos service.RepositoryService, workflows service.WorkflowService) *GithubActionsBot {
	return &GithubActionsBot{repos: repos, workflows: workflows}
}

func (b *GithubActionsBot) Run(ctx context.Context) error {
	repos, err := b.repos.ListAll(ctx)
	if err != nil {
		return err
	}

	for _, repo := range repos {
		workflows, err := b.workflows.List(ctx, repo.Name)
		if err != nil {
			continue
		}

		for _, wf := range workflows {
			fmt.Println(wf.Name)
			fmt.Printf("%+v\n", wf.Content)
		}
	}

	return nil
}
