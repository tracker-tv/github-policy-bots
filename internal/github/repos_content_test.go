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

func TestGetContentsRaw_File(t *testing.T) {
	ctx := context.Background()

	reposSvc := github.NewMockRepositoriesAdapter(t)

	content := &gh.RepositoryContent{
		Name:    gh.Ptr("README.md"),
		Path:    gh.Ptr("README.md"),
		Content: gh.Ptr("# Hello World"),
	}

	reposSvc.
		EXPECT().
		GetContents(mock.Anything, "org-name", "my-repo", "README.md", mock.Anything).
		Once().
		Return(content, nil, &gh.Response{}, nil)

	client := &client{repositories: reposSvc, org: "org-name"}

	file, dir, resp, err := client.GetContentsRaw(ctx, "my-repo", "README.md")

	assert.NoError(t, err)
	assert.NotNil(t, file)
	assert.Nil(t, dir)
	assert.NotNil(t, resp)
	assert.Equal(t, "README.md", file.GetName())
}

func TestGetContentsRaw_Directory(t *testing.T) {
	ctx := context.Background()

	reposSvc := github.NewMockRepositoriesAdapter(t)

	dirContents := []*gh.RepositoryContent{
		{Name: gh.Ptr("file1.go"), Path: gh.Ptr("src/file1.go")},
		{Name: gh.Ptr("file2.go"), Path: gh.Ptr("src/file2.go")},
	}

	reposSvc.
		EXPECT().
		GetContents(mock.Anything, "org-name", "my-repo", "src", mock.Anything).
		Once().
		Return(nil, dirContents, &gh.Response{}, nil)

	client := &client{repositories: reposSvc, org: "org-name"}

	file, dir, resp, err := client.GetContentsRaw(ctx, "my-repo", "src")

	assert.NoError(t, err)
	assert.Nil(t, file)
	assert.NotNil(t, dir)
	assert.NotNil(t, resp)
	assert.Len(t, dir, 2)
	assert.Equal(t, "file1.go", dir[0].GetName())
	assert.Equal(t, "file2.go", dir[1].GetName())
}

func TestGetContentsRaw_NotFound(t *testing.T) {
	ctx := context.Background()

	reposSvc := github.NewMockRepositoriesAdapter(t)

	reposSvc.
		EXPECT().
		GetContents(mock.Anything, "org-name", "my-repo", "nonexistent.txt", mock.Anything).
		Once().
		Return(nil, nil, nil, errors.New("not found"))

	client := &client{repositories: reposSvc, org: "org-name"}

	file, dir, resp, err := client.GetContentsRaw(ctx, "my-repo", "nonexistent.txt")

	assert.Error(t, err)
	assert.Nil(t, file)
	assert.Nil(t, dir)
	assert.Nil(t, resp)
}
