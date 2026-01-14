package orchestrator

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	serviceMocks "github.com/tracker-tv/github-policy-bots/internal/service/mocks"
	"github.com/tracker-tv/github-policy-bots/models"
)

func TestNewGithubActionsBot(t *testing.T) {
	repoSvc := serviceMocks.NewMockRepositoryService(t)
	policySvc := serviceMocks.NewMockPolicyService(t)

	bot := NewGithubActionsBot(repoSvc, policySvc)

	assert.NotNil(t, bot)
	assert.Equal(t, repoSvc, bot.repos)
	assert.Equal(t, policySvc, bot.policy)
}

func TestRun_Success(t *testing.T) {
	ctx := context.Background()
	repoSvc := serviceMocks.NewMockRepositoryService(t)
	policySvc := serviceMocks.NewMockPolicyService(t)

	repos := []models.Repository{
		{Name: "repo1", FullName: "org/repo1", Archived: false},
		{Name: "repo2", FullName: "org/repo2", Archived: false},
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
		Return([]models.PolicyViolation{
			{Repository: repos[0], Policy: models.PolicyWorkflow{Name: "dockerfile"}, Action: models.PolicyActionCreate},
		}, nil)

	policySvc.
		EXPECT().
		Ensure(mock.Anything, repos[1], []string{"README.md"}).
		Once().
		Return([]models.PolicyViolation{}, nil)

	bot := NewGithubActionsBot(repoSvc, policySvc)
	violations, err := bot.Run(ctx)

	assert.NoError(t, err)
	assert.Len(t, violations, 1)
	assert.Equal(t, "dockerfile", violations[0].Policy.Name)
	assert.Equal(t, "repo1", violations[0].Repository.Name)
}

func TestRun_SkipsArchivedRepos(t *testing.T) {
	ctx := context.Background()
	repoSvc := serviceMocks.NewMockRepositoryService(t)
	policySvc := serviceMocks.NewMockPolicyService(t)

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
		Return([]models.PolicyViolation{}, nil)

	bot := NewGithubActionsBot(repoSvc, policySvc)
	violations, err := bot.Run(ctx)

	assert.NoError(t, err)
	assert.Empty(t, violations)
}

func TestRun_ListAllError(t *testing.T) {
	ctx := context.Background()
	repoSvc := serviceMocks.NewMockRepositoryService(t)
	policySvc := serviceMocks.NewMockPolicyService(t)

	repoSvc.
		EXPECT().
		ListAll(mock.Anything).
		Once().
		Return(nil, errors.New("API error"))

	bot := NewGithubActionsBot(repoSvc, policySvc)
	violations, err := bot.Run(ctx)

	assert.Error(t, err)
	assert.Nil(t, violations)
	assert.Contains(t, err.Error(), "API error")
}

func TestRun_ListFilesErrorContinues(t *testing.T) {
	ctx := context.Background()
	repoSvc := serviceMocks.NewMockRepositoryService(t)
	policySvc := serviceMocks.NewMockPolicyService(t)

	repos := []models.Repository{
		{Name: "repo1", FullName: "org/repo1", Archived: false},
		{Name: "repo2", FullName: "org/repo2", Archived: false},
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
		Return([]models.PolicyViolation{
			{Repository: repos[1], Policy: models.PolicyWorkflow{Name: "dockerfile"}, Action: models.PolicyActionCreate},
		}, nil)

	bot := NewGithubActionsBot(repoSvc, policySvc)
	violations, err := bot.Run(ctx)

	assert.NoError(t, err)
	assert.Len(t, violations, 1)
	assert.Equal(t, "repo2", violations[0].Repository.Name)
}

func TestRun_EnsureErrorContinues(t *testing.T) {
	ctx := context.Background()
	repoSvc := serviceMocks.NewMockRepositoryService(t)
	policySvc := serviceMocks.NewMockPolicyService(t)

	repos := []models.Repository{
		{Name: "repo1", FullName: "org/repo1", Archived: false},
		{Name: "repo2", FullName: "org/repo2", Archived: false},
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
		Return([]models.PolicyViolation{
			{Repository: repos[1], Policy: models.PolicyWorkflow{Name: "dockerfile"}, Action: models.PolicyActionCreate},
		}, nil)

	bot := NewGithubActionsBot(repoSvc, policySvc)
	violations, err := bot.Run(ctx)

	assert.NoError(t, err)
	assert.Len(t, violations, 1)
	assert.Equal(t, "repo2", violations[0].Repository.Name)
}

func TestRun_EmptyRepos(t *testing.T) {
	ctx := context.Background()
	repoSvc := serviceMocks.NewMockRepositoryService(t)
	policySvc := serviceMocks.NewMockPolicyService(t)

	repoSvc.
		EXPECT().
		ListAll(mock.Anything).
		Once().
		Return([]models.Repository{}, nil)

	bot := NewGithubActionsBot(repoSvc, policySvc)
	violations, err := bot.Run(ctx)

	assert.NoError(t, err)
	assert.Empty(t, violations)
}

func TestRun_MultipleViolationsAcrossRepos(t *testing.T) {
	ctx := context.Background()
	repoSvc := serviceMocks.NewMockRepositoryService(t)
	policySvc := serviceMocks.NewMockPolicyService(t)

	repos := []models.Repository{
		{Name: "repo1", FullName: "org/repo1", Archived: false},
		{Name: "repo2", FullName: "org/repo2", Archived: false},
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
		Return([]models.PolicyViolation{
			{Repository: repos[0], Policy: models.PolicyWorkflow{Name: "dockerfile"}, Action: models.PolicyActionCreate},
			{Repository: repos[0], Policy: models.PolicyWorkflow{Name: "go-lint"}, Action: models.PolicyActionUpdate},
		}, nil)

	policySvc.
		EXPECT().
		Ensure(mock.Anything, repos[1], []string{"Dockerfile"}).
		Once().
		Return([]models.PolicyViolation{
			{Repository: repos[1], Policy: models.PolicyWorkflow{Name: "dockerfile"}, Action: models.PolicyActionCreate},
		}, nil)

	bot := NewGithubActionsBot(repoSvc, policySvc)
	violations, err := bot.Run(ctx)

	assert.NoError(t, err)
	assert.Len(t, violations, 3)
}
