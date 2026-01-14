package service

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	gh "github.com/google/go-github/v80/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	githubMocks "github.com/tracker-tv/github-policy-bots/internal/github/mocks"
	"github.com/tracker-tv/github-policy-bots/models"
)

func TestNewPolicyService(t *testing.T) {
	mockClient := githubMocks.NewMockClient(t)
	workflows := []models.PolicyWorkflow{
		{Name: "test", MatchFile: "*.go", Source: "http://example.com"},
	}

	svc := NewPolicyService(workflows, mockClient)

	assert.NotNil(t, svc)
	assert.Implements(t, (*PolicyService)(nil), svc)
}

func TestEnsure_NoMatch(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	workflows := []models.PolicyWorkflow{
		{Name: "dockerfile", MatchFile: "**/Dockerfile*", Source: "http://example.com/wf.yml"},
	}

	repo := models.Repository{Name: "my-repo", FullName: "org/my-repo"}
	repoFiles := []string{"main.go", "go.mod", "README.md"}

	svc := NewPolicyService(workflows, mockClient)
	violations, err := svc.Ensure(ctx, repo, repoFiles)

	assert.NoError(t, err)
	assert.Empty(t, violations)
}

func TestEnsure_MatchWithMissingWorkflow(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	workflows := []models.PolicyWorkflow{
		{Name: "dockerfile", MatchFile: "**/Dockerfile*", Source: "http://example.com/wf.yml"},
	}

	repo := models.Repository{Name: "my-repo", FullName: "org/my-repo"}
	repoFiles := []string{"Dockerfile", "main.go"}

	mockClient.
		EXPECT().
		GetContentsRaw(mock.Anything, "my-repo", ".github/workflows/dockerfile.yml").
		Once().
		Return(nil, nil, &gh.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, errors.New("not found"))

	svc := NewPolicyService(workflows, mockClient)
	violations, err := svc.Ensure(ctx, repo, repoFiles)

	assert.NoError(t, err)
	assert.Len(t, violations, 1)
	assert.Equal(t, models.PolicyActionCreate, violations[0].Action)
	assert.Equal(t, ".github/workflows/dockerfile.yml", violations[0].TargetPath)
	assert.Equal(t, "dockerfile", violations[0].Policy.Name)
	assert.Empty(t, violations[0].CurrentContent)
}

func TestEnsure_MatchWithOutdatedWorkflow(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("new workflow content"))
	}))
	defer server.Close()

	workflows := []models.PolicyWorkflow{
		{Name: "dockerfile", MatchFile: "**/Dockerfile*", Source: server.URL},
	}

	repo := models.Repository{Name: "my-repo", FullName: "org/my-repo"}
	repoFiles := []string{"Dockerfile"}

	oldContent := base64.StdEncoding.EncodeToString([]byte("old workflow content"))
	content := &gh.RepositoryContent{
		Content:  gh.Ptr(oldContent),
		Encoding: gh.Ptr("base64"),
	}

	mockClient.
		EXPECT().
		GetContentsRaw(mock.Anything, "my-repo", ".github/workflows/dockerfile.yml").
		Once().
		Return(content, nil, &gh.Response{Response: &http.Response{StatusCode: http.StatusOK}}, nil)

	svc := NewPolicyService(workflows, mockClient)
	violations, err := svc.Ensure(ctx, repo, repoFiles)

	assert.NoError(t, err)
	assert.Len(t, violations, 1)
	assert.Equal(t, models.PolicyActionUpdate, violations[0].Action)
	assert.Equal(t, ".github/workflows/dockerfile.yml", violations[0].TargetPath)
	assert.Equal(t, "old workflow content", violations[0].CurrentContent)
}

func TestEnsure_MatchWithUpToDateWorkflow(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	expectedContent := "current workflow content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedContent))
	}))
	defer server.Close()

	workflows := []models.PolicyWorkflow{
		{Name: "dockerfile", MatchFile: "**/Dockerfile*", Source: server.URL},
	}

	repo := models.Repository{Name: "my-repo", FullName: "org/my-repo"}
	repoFiles := []string{"Dockerfile"}

	encodedContent := base64.StdEncoding.EncodeToString([]byte(expectedContent))
	content := &gh.RepositoryContent{
		Content:  gh.Ptr(encodedContent),
		Encoding: gh.Ptr("base64"),
	}

	mockClient.
		EXPECT().
		GetContentsRaw(mock.Anything, "my-repo", ".github/workflows/dockerfile.yml").
		Once().
		Return(content, nil, &gh.Response{Response: &http.Response{StatusCode: http.StatusOK}}, nil)

	svc := NewPolicyService(workflows, mockClient)
	violations, err := svc.Ensure(ctx, repo, repoFiles)

	assert.NoError(t, err)
	assert.Empty(t, violations)
}

func TestEnsure_MultiplePolcies(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("expected content"))
	}))
	defer server.Close()

	workflows := []models.PolicyWorkflow{
		{Name: "dockerfile", MatchFile: "**/Dockerfile*", Source: server.URL},
		{Name: "go-lint", MatchFile: "go.mod", Source: server.URL},
		{Name: "python", MatchFile: "**/*.py", Source: server.URL},
	}

	repo := models.Repository{Name: "my-repo", FullName: "org/my-repo"}
	repoFiles := []string{"Dockerfile", "go.mod", "utils/helper.go"}

	// dockerfile workflow missing
	mockClient.
		EXPECT().
		GetContentsRaw(mock.Anything, "my-repo", ".github/workflows/dockerfile.yml").
		Once().
		Return(nil, nil, &gh.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, errors.New("not found"))

	// go-lint workflow missing
	mockClient.
		EXPECT().
		GetContentsRaw(mock.Anything, "my-repo", ".github/workflows/go-lint.yml").
		Once().
		Return(nil, nil, &gh.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, errors.New("not found"))

	svc := NewPolicyService(workflows, mockClient)
	violations, err := svc.Ensure(ctx, repo, repoFiles)

	assert.NoError(t, err)
	assert.Len(t, violations, 2)
	assert.Equal(t, "dockerfile", violations[0].Policy.Name)
	assert.Equal(t, "go-lint", violations[1].Policy.Name)
}

func TestEnsure_NestedFileMatch(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	workflows := []models.PolicyWorkflow{
		{Name: "dockerfile", MatchFile: "**/Dockerfile*", Source: "http://example.com/wf.yml"},
	}

	repo := models.Repository{Name: "my-repo", FullName: "org/my-repo"}
	repoFiles := []string{"src/docker/Dockerfile.prod", "main.go"}

	mockClient.
		EXPECT().
		GetContentsRaw(mock.Anything, "my-repo", ".github/workflows/dockerfile.yml").
		Once().
		Return(nil, nil, &gh.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, errors.New("not found"))

	svc := NewPolicyService(workflows, mockClient)
	violations, err := svc.Ensure(ctx, repo, repoFiles)

	assert.NoError(t, err)
	assert.Len(t, violations, 1)
	assert.Equal(t, models.PolicyActionCreate, violations[0].Action)
}

func TestEnsure_GetContentsRawError(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	workflows := []models.PolicyWorkflow{
		{Name: "dockerfile", MatchFile: "**/Dockerfile*", Source: "http://example.com/wf.yml"},
	}

	repo := models.Repository{Name: "my-repo", FullName: "org/my-repo"}
	repoFiles := []string{"Dockerfile"}

	mockClient.
		EXPECT().
		GetContentsRaw(mock.Anything, "my-repo", ".github/workflows/dockerfile.yml").
		Once().
		Return(nil, nil, &gh.Response{Response: &http.Response{StatusCode: http.StatusInternalServerError}}, errors.New("internal server error"))

	svc := NewPolicyService(workflows, mockClient)
	violations, err := svc.Ensure(ctx, repo, repoFiles)

	assert.Error(t, err)
	assert.Nil(t, violations)
	assert.Contains(t, err.Error(), "getting workflow")
}

func TestEnsure_FetchExpectedContentError(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	workflows := []models.PolicyWorkflow{
		{Name: "dockerfile", MatchFile: "**/Dockerfile*", Source: server.URL},
	}

	repo := models.Repository{Name: "my-repo", FullName: "org/my-repo"}
	repoFiles := []string{"Dockerfile"}

	encodedContent := base64.StdEncoding.EncodeToString([]byte("some content"))
	content := &gh.RepositoryContent{
		Content:  gh.Ptr(encodedContent),
		Encoding: gh.Ptr("base64"),
	}

	mockClient.
		EXPECT().
		GetContentsRaw(mock.Anything, "my-repo", ".github/workflows/dockerfile.yml").
		Once().
		Return(content, nil, &gh.Response{Response: &http.Response{StatusCode: http.StatusOK}}, nil)

	svc := NewPolicyService(workflows, mockClient)
	violations, err := svc.Ensure(ctx, repo, repoFiles)

	assert.Error(t, err)
	assert.Nil(t, violations)
	assert.Contains(t, err.Error(), "fetching expected content")
}

func TestEnsure_EmptyRepoFiles(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	workflows := []models.PolicyWorkflow{
		{Name: "dockerfile", MatchFile: "**/Dockerfile*", Source: "http://example.com/wf.yml"},
	}

	repo := models.Repository{Name: "empty-repo", FullName: "org/empty-repo"}
	var repoFiles []string

	svc := NewPolicyService(workflows, mockClient)
	violations, err := svc.Ensure(ctx, repo, repoFiles)

	assert.NoError(t, err)
	assert.Empty(t, violations)
}

func TestEnsure_NoWorkflows(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	var workflows []models.PolicyWorkflow

	repo := models.Repository{Name: "my-repo", FullName: "org/my-repo"}
	repoFiles := []string{"Dockerfile", "main.go"}

	svc := NewPolicyService(workflows, mockClient)
	violations, err := svc.Ensure(ctx, repo, repoFiles)

	assert.NoError(t, err)
	assert.Empty(t, violations)
}
