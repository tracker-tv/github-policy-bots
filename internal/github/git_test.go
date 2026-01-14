package github

import (
	"context"
	"errors"
	"testing"

	gh "github.com/google/go-github/v80/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	github "github.com/tracker-tv/github-policy-bots/internal/github/mocks"
)

func TestGetTree_Success(t *testing.T) {
	ctx := context.Background()

	gitSvc := github.NewMockGitAdapter(t)

	tree := &gh.Tree{
		SHA: gh.Ptr("abc123"),
		Entries: []*gh.TreeEntry{
			{Path: gh.Ptr("README.md"), Type: gh.Ptr("blob")},
			{Path: gh.Ptr("src"), Type: gh.Ptr("tree")},
			{Path: gh.Ptr("src/main.go"), Type: gh.Ptr("blob")},
		},
	}

	gitSvc.
		EXPECT().
		GetTree(mock.Anything, "org-name", "my-repo", "HEAD", true).
		Once().
		Return(tree, &gh.Response{}, nil)

	c := &client{git: gitSvc, org: "org-name"}

	result, resp, err := c.GetTree(ctx, "my-repo", "HEAD", true)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, resp)
	assert.Equal(t, "abc123", result.GetSHA())
	assert.Len(t, result.Entries, 3)
	assert.Equal(t, "README.md", result.Entries[0].GetPath())
}

func TestGetTree_NonRecursive(t *testing.T) {
	ctx := context.Background()

	gitSvc := github.NewMockGitAdapter(t)

	tree := &gh.Tree{
		SHA: gh.Ptr("def456"),
		Entries: []*gh.TreeEntry{
			{Path: gh.Ptr("README.md"), Type: gh.Ptr("blob")},
			{Path: gh.Ptr("src"), Type: gh.Ptr("tree")},
		},
	}

	gitSvc.
		EXPECT().
		GetTree(mock.Anything, "org-name", "my-repo", "main", false).
		Once().
		Return(tree, &gh.Response{}, nil)

	c := &client{git: gitSvc, org: "org-name"}

	result, resp, err := c.GetTree(ctx, "my-repo", "main", false)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, resp)
	assert.Len(t, result.Entries, 2)
}

func TestGetTree_Error(t *testing.T) {
	ctx := context.Background()

	gitSvc := github.NewMockGitAdapter(t)

	gitSvc.
		EXPECT().
		GetTree(mock.Anything, "org-name", "my-repo", "HEAD", true).
		Once().
		Return(nil, nil, errors.New("repository not found"))

	c := &client{git: gitSvc, org: "org-name"}

	result, resp, err := c.GetTree(ctx, "my-repo", "HEAD", true)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "repository not found")
}

func TestGetTree_EmptyRepo(t *testing.T) {
	ctx := context.Background()

	gitSvc := github.NewMockGitAdapter(t)

	tree := &gh.Tree{
		SHA:     gh.Ptr("empty123"),
		Entries: []*gh.TreeEntry{},
	}

	gitSvc.
		EXPECT().
		GetTree(mock.Anything, "org-name", "empty-repo", "HEAD", true).
		Once().
		Return(tree, &gh.Response{}, nil)

	c := &client{git: gitSvc, org: "org-name"}

	result, resp, err := c.GetTree(ctx, "empty-repo", "HEAD", true)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, resp)
	assert.Empty(t, result.Entries)
}
