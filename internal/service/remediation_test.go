package service

import (
	"context"
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

func TestNewRemediationService(t *testing.T) {
	mockClient := githubMocks.NewMockClient(t)

	svc := NewRemediationService(mockClient)

	assert.NotNil(t, svc)
	assert.Implements(t, (*RemediationService)(nil), svc)
}

func TestRemediate_CreateNewPR_Success(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	expectedContent := "workflow content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedContent))
	}))
	defer server.Close()

	drift := models.PolicyDeviation{
		Repository:     models.Repository{Name: "my-repo", FullName: "org/my-repo"},
		Policy:         models.PolicyWorkflow{Name: "dockerfile"},
		Action:         models.PolicyActionCreate,
		TargetPath:     ".github/workflows/dockerfile.yml",
		ExpectedSource: server.URL,
	}

	// No existing PR
	mockClient.
		EXPECT().
		FindPullRequestByBranch(mock.Anything, "my-repo", "chore/dockerfile").
		Once().
		Return(nil, nil)

	// Get main branch
	mockClient.
		EXPECT().
		GetBranch(mock.Anything, "my-repo", "main").
		Once().
		Return(&gh.Reference{
			Ref:    gh.Ptr("refs/heads/main"),
			Object: &gh.GitObject{SHA: gh.Ptr("base-sha-123")},
		}, nil)

	// Create branch
	mockClient.
		EXPECT().
		CreateBranch(mock.Anything, "my-repo", "chore/dockerfile", "base-sha-123").
		Once().
		Return(nil)

	// Verify branch exists
	mockClient.
		EXPECT().
		GetBranch(mock.Anything, "my-repo", "chore/dockerfile").
		Once().
		Return(&gh.Reference{
			Ref:    gh.Ptr("refs/heads/chore/dockerfile"),
			Object: &gh.GitObject{SHA: gh.Ptr("base-sha-123")},
		}, nil)

	// Create file
	mockClient.
		EXPECT().
		CreateOrUpdateFile(mock.Anything, "my-repo", ".github/workflows/dockerfile.yml", "chore/dockerfile", mock.Anything, mock.Anything, mock.Anything).
		Once().
		Return(nil)

	// Create PR
	mockClient.
		EXPECT().
		CreatePullRequest(mock.Anything, "my-repo", mock.Anything, mock.Anything, "chore/dockerfile", "main").
		Once().
		Return(&gh.PullRequest{
			Number:  gh.Ptr(42),
			HTMLURL: gh.Ptr("https://github.com/org/my-repo/pull/42"),
		}, nil)

	svc := NewRemediationService(mockClient)
	result, err := svc.Remediate(ctx, drift)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "created", result.Action)
	assert.Equal(t, "https://github.com/org/my-repo/pull/42", result.PRURL)
}

func TestRemediate_ExistingPR_ContentMatches_Skip(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	expectedContent := "workflow content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedContent))
	}))
	defer server.Close()

	drift := models.PolicyDeviation{
		Repository:     models.Repository{Name: "my-repo", FullName: "org/my-repo"},
		Policy:         models.PolicyWorkflow{Name: "dockerfile"},
		Action:         models.PolicyActionUpdate,
		TargetPath:     ".github/workflows/dockerfile.yml",
		ExpectedSource: server.URL,
	}

	existingPR := &gh.PullRequest{
		Number:  gh.Ptr(10),
		HTMLURL: gh.Ptr("https://github.com/org/my-repo/pull/10"),
	}

	// PR already exists
	mockClient.
		EXPECT().
		FindPullRequestByBranch(mock.Anything, "my-repo", "chore/dockerfile").
		Once().
		Return(existingPR, nil)

	// Get current content on branch - matches expected (wrapped)
	wrappedContent := wrapContent(expectedContent, "dockerfile")
	mockClient.
		EXPECT().
		GetFileContent(mock.Anything, "my-repo", ".github/workflows/dockerfile.yml", "chore/dockerfile").
		Once().
		Return(wrappedContent, "existing-sha", nil)

	svc := NewRemediationService(mockClient)
	result, err := svc.Remediate(ctx, drift)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "skipped", result.Action)
	assert.Equal(t, "https://github.com/org/my-repo/pull/10", result.PRURL)
}

func TestRemediate_ExistingPR_ContentDiffers_Update(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	expectedContent := "new workflow content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedContent))
	}))
	defer server.Close()

	drift := models.PolicyDeviation{
		Repository:     models.Repository{Name: "my-repo", FullName: "org/my-repo"},
		Policy:         models.PolicyWorkflow{Name: "dockerfile"},
		Action:         models.PolicyActionUpdate,
		TargetPath:     ".github/workflows/dockerfile.yml",
		ExpectedSource: server.URL,
	}

	existingPR := &gh.PullRequest{
		Number:  gh.Ptr(10),
		HTMLURL: gh.Ptr("https://github.com/org/my-repo/pull/10"),
	}

	// PR already exists
	mockClient.
		EXPECT().
		FindPullRequestByBranch(mock.Anything, "my-repo", "chore/dockerfile").
		Once().
		Return(existingPR, nil)

	// Get current content on branch - differs from expected
	mockClient.
		EXPECT().
		GetFileContent(mock.Anything, "my-repo", ".github/workflows/dockerfile.yml", "chore/dockerfile").
		Once().
		Return("old content", "existing-sha", nil)

	// Update file
	mockClient.
		EXPECT().
		CreateOrUpdateFile(mock.Anything, "my-repo", ".github/workflows/dockerfile.yml", "chore/dockerfile", mock.Anything, mock.Anything, mock.Anything).
		Once().
		Return(nil)

	svc := NewRemediationService(mockClient)
	result, err := svc.Remediate(ctx, drift)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "updated", result.Action)
	assert.Equal(t, "https://github.com/org/my-repo/pull/10", result.PRURL)
}

func TestRemediate_FetchContentError(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	drift := models.PolicyDeviation{
		Repository:     models.Repository{Name: "my-repo", FullName: "org/my-repo"},
		Policy:         models.PolicyWorkflow{Name: "dockerfile"},
		Action:         models.PolicyActionCreate,
		TargetPath:     ".github/workflows/dockerfile.yml",
		ExpectedSource: server.URL,
	}

	svc := NewRemediationService(mockClient)
	result, err := svc.Remediate(ctx, drift)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "fetching expected content")
}

func TestRemediate_FindPRError(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	expectedContent := "workflow content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedContent))
	}))
	defer server.Close()

	drift := models.PolicyDeviation{
		Repository:     models.Repository{Name: "my-repo", FullName: "org/my-repo"},
		Policy:         models.PolicyWorkflow{Name: "dockerfile"},
		Action:         models.PolicyActionCreate,
		TargetPath:     ".github/workflows/dockerfile.yml",
		ExpectedSource: server.URL,
	}

	mockClient.
		EXPECT().
		FindPullRequestByBranch(mock.Anything, "my-repo", "chore/dockerfile").
		Once().
		Return(nil, errors.New("API error"))

	svc := NewRemediationService(mockClient)
	result, err := svc.Remediate(ctx, drift)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "finding existing PR")
}

func TestRemediate_GetDefaultBranchError(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	expectedContent := "workflow content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedContent))
	}))
	defer server.Close()

	drift := models.PolicyDeviation{
		Repository:     models.Repository{Name: "my-repo", FullName: "org/my-repo"},
		Policy:         models.PolicyWorkflow{Name: "dockerfile"},
		Action:         models.PolicyActionCreate,
		TargetPath:     ".github/workflows/dockerfile.yml",
		ExpectedSource: server.URL,
	}

	mockClient.
		EXPECT().
		FindPullRequestByBranch(mock.Anything, "my-repo", "chore/dockerfile").
		Once().
		Return(nil, nil)

	// Main branch not found
	mockClient.
		EXPECT().
		GetBranch(mock.Anything, "my-repo", "main").
		Once().
		Return(nil, errors.New("not found"))

	// Master branch not found either
	mockClient.
		EXPECT().
		GetBranch(mock.Anything, "my-repo", "master").
		Once().
		Return(nil, errors.New("not found"))

	svc := NewRemediationService(mockClient)
	result, err := svc.Remediate(ctx, drift)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "getting default branch")
}

func TestRemediate_CreateBranchError(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	expectedContent := "workflow content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedContent))
	}))
	defer server.Close()

	drift := models.PolicyDeviation{
		Repository:     models.Repository{Name: "my-repo", FullName: "org/my-repo"},
		Policy:         models.PolicyWorkflow{Name: "dockerfile"},
		Action:         models.PolicyActionCreate,
		TargetPath:     ".github/workflows/dockerfile.yml",
		ExpectedSource: server.URL,
	}

	mockClient.
		EXPECT().
		FindPullRequestByBranch(mock.Anything, "my-repo", "chore/dockerfile").
		Once().
		Return(nil, nil)

	mockClient.
		EXPECT().
		GetBranch(mock.Anything, "my-repo", "main").
		Once().
		Return(&gh.Reference{
			Ref:    gh.Ptr("refs/heads/main"),
			Object: &gh.GitObject{SHA: gh.Ptr("base-sha-123")},
		}, nil)

	// Branch creation fails
	mockClient.
		EXPECT().
		CreateBranch(mock.Anything, "my-repo", "chore/dockerfile", "base-sha-123").
		Once().
		Return(errors.New("permission denied"))

	// Branch doesn't exist (not a pre-existing branch)
	mockClient.
		EXPECT().
		GetBranch(mock.Anything, "my-repo", "chore/dockerfile").
		Once().
		Return(nil, errors.New("not found"))

	svc := NewRemediationService(mockClient)
	result, err := svc.Remediate(ctx, drift)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "creating branch")
}

func TestRemediate_CreateFileError(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	expectedContent := "workflow content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedContent))
	}))
	defer server.Close()

	drift := models.PolicyDeviation{
		Repository:     models.Repository{Name: "my-repo", FullName: "org/my-repo"},
		Policy:         models.PolicyWorkflow{Name: "dockerfile"},
		Action:         models.PolicyActionCreate,
		TargetPath:     ".github/workflows/dockerfile.yml",
		ExpectedSource: server.URL,
	}

	mockClient.
		EXPECT().
		FindPullRequestByBranch(mock.Anything, "my-repo", "chore/dockerfile").
		Once().
		Return(nil, nil)

	mockClient.
		EXPECT().
		GetBranch(mock.Anything, "my-repo", "main").
		Once().
		Return(&gh.Reference{
			Ref:    gh.Ptr("refs/heads/main"),
			Object: &gh.GitObject{SHA: gh.Ptr("base-sha-123")},
		}, nil)

	mockClient.
		EXPECT().
		CreateBranch(mock.Anything, "my-repo", "chore/dockerfile", "base-sha-123").
		Once().
		Return(nil)

	mockClient.
		EXPECT().
		GetBranch(mock.Anything, "my-repo", "chore/dockerfile").
		Once().
		Return(&gh.Reference{
			Ref:    gh.Ptr("refs/heads/chore/dockerfile"),
			Object: &gh.GitObject{SHA: gh.Ptr("base-sha-123")},
		}, nil)

	mockClient.
		EXPECT().
		CreateOrUpdateFile(mock.Anything, "my-repo", ".github/workflows/dockerfile.yml", "chore/dockerfile", mock.Anything, mock.Anything, mock.Anything).
		Once().
		Return(errors.New("404 not found"))

	svc := NewRemediationService(mockClient)
	result, err := svc.Remediate(ctx, drift)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "creating file")
}

func TestRemediate_CreatePRError(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	expectedContent := "workflow content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedContent))
	}))
	defer server.Close()

	drift := models.PolicyDeviation{
		Repository:     models.Repository{Name: "my-repo", FullName: "org/my-repo"},
		Policy:         models.PolicyWorkflow{Name: "dockerfile"},
		Action:         models.PolicyActionCreate,
		TargetPath:     ".github/workflows/dockerfile.yml",
		ExpectedSource: server.URL,
	}

	mockClient.
		EXPECT().
		FindPullRequestByBranch(mock.Anything, "my-repo", "chore/dockerfile").
		Once().
		Return(nil, nil)

	mockClient.
		EXPECT().
		GetBranch(mock.Anything, "my-repo", "main").
		Once().
		Return(&gh.Reference{
			Ref:    gh.Ptr("refs/heads/main"),
			Object: &gh.GitObject{SHA: gh.Ptr("base-sha-123")},
		}, nil)

	mockClient.
		EXPECT().
		CreateBranch(mock.Anything, "my-repo", "chore/dockerfile", "base-sha-123").
		Once().
		Return(nil)

	mockClient.
		EXPECT().
		GetBranch(mock.Anything, "my-repo", "chore/dockerfile").
		Once().
		Return(&gh.Reference{
			Ref:    gh.Ptr("refs/heads/chore/dockerfile"),
			Object: &gh.GitObject{SHA: gh.Ptr("base-sha-123")},
		}, nil)

	mockClient.
		EXPECT().
		CreateOrUpdateFile(mock.Anything, "my-repo", ".github/workflows/dockerfile.yml", "chore/dockerfile", mock.Anything, mock.Anything, mock.Anything).
		Once().
		Return(nil)

	mockClient.
		EXPECT().
		CreatePullRequest(mock.Anything, "my-repo", mock.Anything, mock.Anything, "chore/dockerfile", "main").
		Once().
		Return(nil, errors.New("PR already exists"))

	svc := NewRemediationService(mockClient)
	result, err := svc.Remediate(ctx, drift)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "creating PR")
}

func TestRemediate_FallbackToMasterBranch(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	expectedContent := "workflow content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedContent))
	}))
	defer server.Close()

	drift := models.PolicyDeviation{
		Repository:     models.Repository{Name: "my-repo", FullName: "org/my-repo"},
		Policy:         models.PolicyWorkflow{Name: "dockerfile"},
		Action:         models.PolicyActionCreate,
		TargetPath:     ".github/workflows/dockerfile.yml",
		ExpectedSource: server.URL,
	}

	mockClient.
		EXPECT().
		FindPullRequestByBranch(mock.Anything, "my-repo", "chore/dockerfile").
		Once().
		Return(nil, nil)

	// Main not found
	mockClient.
		EXPECT().
		GetBranch(mock.Anything, "my-repo", "main").
		Once().
		Return(nil, errors.New("not found"))

	// Fallback to master
	mockClient.
		EXPECT().
		GetBranch(mock.Anything, "my-repo", "master").
		Once().
		Return(&gh.Reference{
			Ref:    gh.Ptr("refs/heads/master"),
			Object: &gh.GitObject{SHA: gh.Ptr("master-sha-456")},
		}, nil)

	mockClient.
		EXPECT().
		CreateBranch(mock.Anything, "my-repo", "chore/dockerfile", "master-sha-456").
		Once().
		Return(nil)

	mockClient.
		EXPECT().
		GetBranch(mock.Anything, "my-repo", "chore/dockerfile").
		Once().
		Return(&gh.Reference{
			Ref:    gh.Ptr("refs/heads/chore/dockerfile"),
			Object: &gh.GitObject{SHA: gh.Ptr("master-sha-456")},
		}, nil)

	mockClient.
		EXPECT().
		CreateOrUpdateFile(mock.Anything, "my-repo", ".github/workflows/dockerfile.yml", "chore/dockerfile", mock.Anything, mock.Anything, mock.Anything).
		Once().
		Return(nil)

	mockClient.
		EXPECT().
		CreatePullRequest(mock.Anything, "my-repo", mock.Anything, mock.Anything, "chore/dockerfile", "main").
		Once().
		Return(&gh.PullRequest{
			Number:  gh.Ptr(1),
			HTMLURL: gh.Ptr("https://github.com/org/my-repo/pull/1"),
		}, nil)

	svc := NewRemediationService(mockClient)
	result, err := svc.Remediate(ctx, drift)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "created", result.Action)
}

func TestRemediate_BranchAlreadyExists_Continue(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	expectedContent := "workflow content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedContent))
	}))
	defer server.Close()

	drift := models.PolicyDeviation{
		Repository:     models.Repository{Name: "my-repo", FullName: "org/my-repo"},
		Policy:         models.PolicyWorkflow{Name: "dockerfile"},
		Action:         models.PolicyActionCreate,
		TargetPath:     ".github/workflows/dockerfile.yml",
		ExpectedSource: server.URL,
	}

	mockClient.
		EXPECT().
		FindPullRequestByBranch(mock.Anything, "my-repo", "chore/dockerfile").
		Once().
		Return(nil, nil)

	mockClient.
		EXPECT().
		GetBranch(mock.Anything, "my-repo", "main").
		Once().
		Return(&gh.Reference{
			Ref:    gh.Ptr("refs/heads/main"),
			Object: &gh.GitObject{SHA: gh.Ptr("base-sha-123")},
		}, nil)

	// Branch creation fails because it already exists
	mockClient.
		EXPECT().
		CreateBranch(mock.Anything, "my-repo", "chore/dockerfile", "base-sha-123").
		Once().
		Return(errors.New("reference already exists"))

	// But branch exists when we check
	mockClient.
		EXPECT().
		GetBranch(mock.Anything, "my-repo", "chore/dockerfile").
		Times(2). // Once for check after error, once for verification
		Return(&gh.Reference{
			Ref:    gh.Ptr("refs/heads/chore/dockerfile"),
			Object: &gh.GitObject{SHA: gh.Ptr("base-sha-123")},
		}, nil)

	mockClient.
		EXPECT().
		CreateOrUpdateFile(mock.Anything, "my-repo", ".github/workflows/dockerfile.yml", "chore/dockerfile", mock.Anything, mock.Anything, mock.Anything).
		Once().
		Return(nil)

	mockClient.
		EXPECT().
		CreatePullRequest(mock.Anything, "my-repo", mock.Anything, mock.Anything, "chore/dockerfile", "main").
		Once().
		Return(&gh.PullRequest{
			Number:  gh.Ptr(1),
			HTMLURL: gh.Ptr("https://github.com/org/my-repo/pull/1"),
		}, nil)

	svc := NewRemediationService(mockClient)
	result, err := svc.Remediate(ctx, drift)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "created", result.Action)
}

func TestWrapContent(t *testing.T) {
	content := "name: test\non: push"
	policyName := "dockerfile"

	wrapped := wrapContent(content, policyName)

	assert.Contains(t, wrapped, "# DO NOT EDIT: BEGIN")
	assert.Contains(t, wrapped, "# DO NOT EDIT: END")
	assert.Contains(t, wrapped, "tracker-tv-bot")
	assert.Contains(t, wrapped, policyName)
	assert.Contains(t, wrapped, content)
}

func TestActionVerb(t *testing.T) {
	assert.Equal(t, "add", actionVerb(models.PolicyActionCreate))
	assert.Equal(t, "update", actionVerb(models.PolicyActionUpdate))
}
