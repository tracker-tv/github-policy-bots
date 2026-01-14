package service

import (
	"context"
	"errors"
	"testing"

	gh "github.com/google/go-github/v80/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	githubMocks "github.com/tracker-tv/github-policy-bots/internal/github/mocks"
)

func TestNewRepositoriesService(t *testing.T) {
	mockClient := githubMocks.NewMockClient(t)

	svc := NewRepositoriesService(mockClient)

	assert.NotNil(t, svc)
	assert.Implements(t, (*RepositoryService)(nil), svc)
}

func TestListAll_Success(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	repos := []*gh.Repository{
		{
			Name:     gh.Ptr("repo1"),
			FullName: gh.Ptr("org/repo1"),
			Private:  gh.Ptr(false),
			Archived: gh.Ptr(false),
		},
		{
			Name:     gh.Ptr("repo2"),
			FullName: gh.Ptr("org/repo2"),
			Private:  gh.Ptr(true),
			Archived: gh.Ptr(true),
		},
	}

	mockClient.
		EXPECT().
		ListAllRepos(mock.Anything).
		Once().
		Return(repos, nil)

	svc := NewRepositoriesService(mockClient)
	result, err := svc.ListAll(ctx)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "repo1", result[0].Name)
	assert.Equal(t, "org/repo1", result[0].FullName)
	assert.False(t, result[0].Private)
	assert.False(t, result[0].Archived)
	assert.Equal(t, "repo2", result[1].Name)
	assert.Equal(t, "org/repo2", result[1].FullName)
	assert.True(t, result[1].Private)
	assert.True(t, result[1].Archived)
}

func TestListAll_WithNilRepo(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	repos := []*gh.Repository{
		{
			Name:     gh.Ptr("repo1"),
			FullName: gh.Ptr("org/repo1"),
			Private:  gh.Ptr(false),
			Archived: gh.Ptr(false),
		},
		nil,
		{
			Name:     gh.Ptr("repo2"),
			FullName: gh.Ptr("org/repo2"),
			Private:  gh.Ptr(false),
			Archived: gh.Ptr(false),
		},
	}

	mockClient.
		EXPECT().
		ListAllRepos(mock.Anything).
		Once().
		Return(repos, nil)

	svc := NewRepositoriesService(mockClient)
	result, err := svc.ListAll(ctx)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "repo1", result[0].Name)
	assert.Equal(t, "repo2", result[1].Name)
}

func TestListAll_Empty(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	mockClient.
		EXPECT().
		ListAllRepos(mock.Anything).
		Once().
		Return([]*gh.Repository{}, nil)

	svc := NewRepositoriesService(mockClient)
	result, err := svc.ListAll(ctx)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestListAll_Error(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	mockClient.
		EXPECT().
		ListAllRepos(mock.Anything).
		Once().
		Return(nil, errors.New("API error"))

	svc := NewRepositoriesService(mockClient)
	result, err := svc.ListAll(ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "API error")
}

func TestListFiles_Success(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	tree := &gh.Tree{
		SHA: gh.Ptr("abc123"),
		Entries: []*gh.TreeEntry{
			{Path: gh.Ptr("README.md"), Type: gh.Ptr("blob")},
			{Path: gh.Ptr("src"), Type: gh.Ptr("tree")},
			{Path: gh.Ptr("src/main.go"), Type: gh.Ptr("blob")},
			{Path: gh.Ptr("src/utils"), Type: gh.Ptr("tree")},
			{Path: gh.Ptr("src/utils/helper.go"), Type: gh.Ptr("blob")},
		},
	}

	mockClient.
		EXPECT().
		GetTree(mock.Anything, "my-repo", "HEAD", true).
		Once().
		Return(tree, &gh.Response{}, nil)

	svc := NewRepositoriesService(mockClient)
	result, err := svc.ListFiles(ctx, "my-repo")

	assert.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Contains(t, result, "README.md")
	assert.Contains(t, result, "src/main.go")
	assert.Contains(t, result, "src/utils/helper.go")
	assert.NotContains(t, result, "src")
	assert.NotContains(t, result, "src/utils")
}

func TestListFiles_EmptyRepo(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	tree := &gh.Tree{
		SHA:     gh.Ptr("empty123"),
		Entries: []*gh.TreeEntry{},
	}

	mockClient.
		EXPECT().
		GetTree(mock.Anything, "empty-repo", "HEAD", true).
		Once().
		Return(tree, &gh.Response{}, nil)

	svc := NewRepositoriesService(mockClient)
	result, err := svc.ListFiles(ctx, "empty-repo")

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestListFiles_Error(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	mockClient.
		EXPECT().
		GetTree(mock.Anything, "my-repo", "HEAD", true).
		Once().
		Return(nil, nil, errors.New("repository not found"))

	svc := NewRepositoriesService(mockClient)
	result, err := svc.ListFiles(ctx, "my-repo")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "repository not found")
}

func TestListFiles_OnlyDirectories(t *testing.T) {
	ctx := context.Background()
	mockClient := githubMocks.NewMockClient(t)

	tree := &gh.Tree{
		SHA: gh.Ptr("dirs123"),
		Entries: []*gh.TreeEntry{
			{Path: gh.Ptr("src"), Type: gh.Ptr("tree")},
			{Path: gh.Ptr("pkg"), Type: gh.Ptr("tree")},
		},
	}

	mockClient.
		EXPECT().
		GetTree(mock.Anything, "dirs-only-repo", "HEAD", true).
		Once().
		Return(tree, &gh.Response{}, nil)

	svc := NewRepositoriesService(mockClient)
	result, err := svc.ListFiles(ctx, "dirs-only-repo")

	assert.NoError(t, err)
	assert.Empty(t, result)
}
