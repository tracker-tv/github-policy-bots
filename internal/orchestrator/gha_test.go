package orchestrator

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tracker-tv/github-policy-bots/internal/service"
	serviceMocks "github.com/tracker-tv/github-policy-bots/internal/service/mocks"
	"github.com/tracker-tv/github-policy-bots/models"
)

func TestNewGithubActionsBot(t *testing.T) {
	repoSvc := serviceMocks.NewMockRepositoryService(t)
	policySvc := serviceMocks.NewMockPolicyService(t)
	remediationSvc := serviceMocks.NewMockRemediationService(t)

	bot := NewGithubActionsBot(repoSvc, policySvc, remediationSvc)

	assert.NotNil(t, bot)
	assert.Equal(t, repoSvc, bot.repos)
	assert.Equal(t, policySvc, bot.policy)
	assert.Equal(t, remediationSvc, bot.remediation)
}

func TestRun_Success(t *testing.T) {
	ctx := context.Background()
	repoSvc := serviceMocks.NewMockRepositoryService(t)
	policySvc := serviceMocks.NewMockPolicyService(t)
	remediationSvc := serviceMocks.NewMockRemediationService(t)

	repos := []models.Repository{
		{Name: "repo1", FullName: "org/repo1", Archived: false},
		{Name: "repo2", FullName: "org/repo2", Archived: false},
	}

	drift := models.PolicyDeviation{
		Repository: repos[0],
		Policy:     models.PolicyWorkflow{Name: "dockerfile"},
		Action:     models.PolicyActionCreate,
	}

	repoSvc.
		EXPECT().
		ListAll(mock.Anything).
		Once().
		Return(repos, nil)

	repoSvc.
		EXPECT().
		ListFiles(mock.Anything, "repo1").
		Once().
		Return([]string{"Dockerfile", "main.go"}, nil)

	repoSvc.
		EXPECT().
		ListFiles(mock.Anything, "repo2").
		Once().
		Return([]string{"README.md"}, nil)

	policySvc.
		EXPECT().
		Ensure(mock.Anything, repos[0], []string{"Dockerfile", "main.go"}).
		Once().
		Return([]models.PolicyDeviation{drift}, nil)

	policySvc.
		EXPECT().
		Ensure(mock.Anything, repos[1], []string{"README.md"}).
		Once().
		Return([]models.PolicyDeviation{}, nil)

	remediationSvc.
		EXPECT().
		Remediate(mock.Anything, drift).
		Once().
		Return(&service.RemediationResult{
			Drift:  drift,
			Action: "created",
			PRURL:  "https://github.com/org/repo1/pull/1",
		}, nil)

	bot := NewGithubActionsBot(repoSvc, policySvc, remediationSvc)
	results, err := bot.Run(ctx)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "dockerfile", results[0].Drift.Policy.Name)
	assert.Equal(t, "repo1", results[0].Drift.Repository.Name)
	assert.Equal(t, "created", results[0].Action)
}

func TestRun_SkipsArchivedRepos(t *testing.T) {
	ctx := context.Background()
	repoSvc := serviceMocks.NewMockRepositoryService(t)
	policySvc := serviceMocks.NewMockPolicyService(t)
	remediationSvc := serviceMocks.NewMockRemediationService(t)

	repos := []models.Repository{
		{Name: "active-repo", FullName: "org/active-repo", Archived: false},
		{Name: "archived-repo", FullName: "org/archived-repo", Archived: true},
	}

	repoSvc.
		EXPECT().
		ListAll(mock.Anything).
		Once().
		Return(repos, nil)

	repoSvc.
		EXPECT().
		ListFiles(mock.Anything, "active-repo").
		Once().
		Return([]string{"main.go"}, nil)

	policySvc.
		EXPECT().
		Ensure(mock.Anything, repos[0], []string{"main.go"}).
		Once().
		Return([]models.PolicyDeviation{}, nil)

	bot := NewGithubActionsBot(repoSvc, policySvc, remediationSvc)
	results, err := bot.Run(ctx)

	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestRun_ListAllError(t *testing.T) {
	ctx := context.Background()
	repoSvc := serviceMocks.NewMockRepositoryService(t)
	policySvc := serviceMocks.NewMockPolicyService(t)
	remediationSvc := serviceMocks.NewMockRemediationService(t)

	repoSvc.
		EXPECT().
		ListAll(mock.Anything).
		Once().
		Return(nil, errors.New("API error"))

	bot := NewGithubActionsBot(repoSvc, policySvc, remediationSvc)
	results, err := bot.Run(ctx)

	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "API error")
}

func TestRun_ListFilesErrorContinues(t *testing.T) {
	ctx := context.Background()
	repoSvc := serviceMocks.NewMockRepositoryService(t)
	policySvc := serviceMocks.NewMockPolicyService(t)
	remediationSvc := serviceMocks.NewMockRemediationService(t)

	repos := []models.Repository{
		{Name: "repo1", FullName: "org/repo1", Archived: false},
		{Name: "repo2", FullName: "org/repo2", Archived: false},
	}

	drift := models.PolicyDeviation{
		Repository: repos[1],
		Policy:     models.PolicyWorkflow{Name: "dockerfile"},
		Action:     models.PolicyActionCreate,
	}

	repoSvc.
		EXPECT().
		ListAll(mock.Anything).
		Once().
		Return(repos, nil)

	repoSvc.
		EXPECT().
		ListFiles(mock.Anything, "repo1").
		Once().
		Return(nil, errors.New("empty repo"))

	repoSvc.
		EXPECT().
		ListFiles(mock.Anything, "repo2").
		Once().
		Return([]string{"Dockerfile"}, nil)

	policySvc.
		EXPECT().
		Ensure(mock.Anything, repos[1], []string{"Dockerfile"}).
		Once().
		Return([]models.PolicyDeviation{drift}, nil)

	remediationSvc.
		EXPECT().
		Remediate(mock.Anything, drift).
		Once().
		Return(&service.RemediationResult{
			Drift:  drift,
			Action: "created",
			PRURL:  "https://github.com/org/repo2/pull/1",
		}, nil)

	bot := NewGithubActionsBot(repoSvc, policySvc, remediationSvc)
	results, err := bot.Run(ctx)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "repo2", results[0].Drift.Repository.Name)
}

func TestRun_EnsureErrorContinues(t *testing.T) {
	ctx := context.Background()
	repoSvc := serviceMocks.NewMockRepositoryService(t)
	policySvc := serviceMocks.NewMockPolicyService(t)
	remediationSvc := serviceMocks.NewMockRemediationService(t)

	repos := []models.Repository{
		{Name: "repo1", FullName: "org/repo1", Archived: false},
		{Name: "repo2", FullName: "org/repo2", Archived: false},
	}

	drift := models.PolicyDeviation{
		Repository: repos[1],
		Policy:     models.PolicyWorkflow{Name: "dockerfile"},
		Action:     models.PolicyActionCreate,
	}

	repoSvc.
		EXPECT().
		ListAll(mock.Anything).
		Once().
		Return(repos, nil)

	repoSvc.
		EXPECT().
		ListFiles(mock.Anything, "repo1").
		Once().
		Return([]string{"Dockerfile"}, nil)

	repoSvc.
		EXPECT().
		ListFiles(mock.Anything, "repo2").
		Once().
		Return([]string{"Dockerfile"}, nil)

	policySvc.
		EXPECT().
		Ensure(mock.Anything, repos[0], []string{"Dockerfile"}).
		Once().
		Return(nil, errors.New("policy check failed"))

	policySvc.
		EXPECT().
		Ensure(mock.Anything, repos[1], []string{"Dockerfile"}).
		Once().
		Return([]models.PolicyDeviation{drift}, nil)

	remediationSvc.
		EXPECT().
		Remediate(mock.Anything, drift).
		Once().
		Return(&service.RemediationResult{
			Drift:  drift,
			Action: "created",
			PRURL:  "https://github.com/org/repo2/pull/1",
		}, nil)

	bot := NewGithubActionsBot(repoSvc, policySvc, remediationSvc)
	results, err := bot.Run(ctx)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "repo2", results[0].Drift.Repository.Name)
}

func TestRun_EmptyRepos(t *testing.T) {
	ctx := context.Background()
	repoSvc := serviceMocks.NewMockRepositoryService(t)
	policySvc := serviceMocks.NewMockPolicyService(t)
	remediationSvc := serviceMocks.NewMockRemediationService(t)

	repoSvc.
		EXPECT().
		ListAll(mock.Anything).
		Once().
		Return([]models.Repository{}, nil)

	bot := NewGithubActionsBot(repoSvc, policySvc, remediationSvc)
	results, err := bot.Run(ctx)

	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestRun_MultipleResultsAcrossRepos(t *testing.T) {
	ctx := context.Background()
	repoSvc := serviceMocks.NewMockRepositoryService(t)
	policySvc := serviceMocks.NewMockPolicyService(t)
	remediationSvc := serviceMocks.NewMockRemediationService(t)

	repos := []models.Repository{
		{Name: "repo1", FullName: "org/repo1", Archived: false},
		{Name: "repo2", FullName: "org/repo2", Archived: false},
	}

	drift1 := models.PolicyDeviation{
		Repository: repos[0],
		Policy:     models.PolicyWorkflow{Name: "dockerfile"},
		Action:     models.PolicyActionCreate,
	}
	drift2 := models.PolicyDeviation{
		Repository: repos[0],
		Policy:     models.PolicyWorkflow{Name: "go-lint"},
		Action:     models.PolicyActionUpdate,
	}
	drift3 := models.PolicyDeviation{
		Repository: repos[1],
		Policy:     models.PolicyWorkflow{Name: "dockerfile"},
		Action:     models.PolicyActionCreate,
	}

	repoSvc.
		EXPECT().
		ListAll(mock.Anything).
		Once().
		Return(repos, nil)

	repoSvc.
		EXPECT().
		ListFiles(mock.Anything, "repo1").
		Once().
		Return([]string{"Dockerfile", "main.go"}, nil)

	repoSvc.
		EXPECT().
		ListFiles(mock.Anything, "repo2").
		Once().
		Return([]string{"Dockerfile"}, nil)

	policySvc.
		EXPECT().
		Ensure(mock.Anything, repos[0], []string{"Dockerfile", "main.go"}).
		Once().
		Return([]models.PolicyDeviation{drift1, drift2}, nil)

	policySvc.
		EXPECT().
		Ensure(mock.Anything, repos[1], []string{"Dockerfile"}).
		Once().
		Return([]models.PolicyDeviation{drift3}, nil)

	remediationSvc.
		EXPECT().
		Remediate(mock.Anything, drift1).
		Once().
		Return(&service.RemediationResult{Drift: drift1, Action: "created", PRURL: "url1"}, nil)

	remediationSvc.
		EXPECT().
		Remediate(mock.Anything, drift2).
		Once().
		Return(&service.RemediationResult{Drift: drift2, Action: "updated", PRURL: "url2"}, nil)

	remediationSvc.
		EXPECT().
		Remediate(mock.Anything, drift3).
		Once().
		Return(&service.RemediationResult{Drift: drift3, Action: "created", PRURL: "url3"}, nil)

	bot := NewGithubActionsBot(repoSvc, policySvc, remediationSvc)
	results, err := bot.Run(ctx)

	assert.NoError(t, err)
	assert.Len(t, results, 3)
}

func TestRun_RemediationErrorContinues(t *testing.T) {
	ctx := context.Background()
	repoSvc := serviceMocks.NewMockRepositoryService(t)
	policySvc := serviceMocks.NewMockPolicyService(t)
	remediationSvc := serviceMocks.NewMockRemediationService(t)

	repos := []models.Repository{
		{Name: "repo1", FullName: "org/repo1", Archived: false},
	}

	drift1 := models.PolicyDeviation{
		Repository: repos[0],
		Policy:     models.PolicyWorkflow{Name: "dockerfile"},
		Action:     models.PolicyActionCreate,
	}
	drift2 := models.PolicyDeviation{
		Repository: repos[0],
		Policy:     models.PolicyWorkflow{Name: "go-lint"},
		Action:     models.PolicyActionUpdate,
	}

	repoSvc.
		EXPECT().
		ListAll(mock.Anything).
		Once().
		Return(repos, nil)

	repoSvc.
		EXPECT().
		ListFiles(mock.Anything, "repo1").
		Once().
		Return([]string{"Dockerfile", "main.go"}, nil)

	policySvc.
		EXPECT().
		Ensure(mock.Anything, repos[0], []string{"Dockerfile", "main.go"}).
		Once().
		Return([]models.PolicyDeviation{drift1, drift2}, nil)

	remediationSvc.
		EXPECT().
		Remediate(mock.Anything, drift1).
		Once().
		Return(nil, errors.New("PR creation failed"))

	remediationSvc.
		EXPECT().
		Remediate(mock.Anything, drift2).
		Once().
		Return(&service.RemediationResult{Drift: drift2, Action: "created", PRURL: "url2"}, nil)

	bot := NewGithubActionsBot(repoSvc, policySvc, remediationSvc)
	results, err := bot.Run(ctx)

	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.NotNil(t, results[0].Error)
	assert.Nil(t, results[1].Error)
	assert.Equal(t, "created", results[1].Action)
}
