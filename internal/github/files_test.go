package github

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	gh "github.com/google/go-github/v80/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	github "github.com/tracker-tv/github-policy-bots/internal/github/mocks"
)

func TestGetFileContent_Success(t *testing.T) {
	ctx := context.Background()
	repoSvc := github.NewMockRepositoriesAdapter(t)

	fileContent := "name: test\non: push"
	encodedContent := base64.StdEncoding.EncodeToString([]byte(fileContent))

	repoSvc.
		EXPECT().
		GetContents(mock.Anything, "org-name", "repo-name", ".github/workflows/test.yml",
			mock.MatchedBy(func(opts *gh.RepositoryContentGetOptions) bool {
				return opts.Ref == "main"
			}),
		).
		Once().
		Return(
			&gh.RepositoryContent{
				Content:  gh.Ptr(encodedContent),
				Encoding: gh.Ptr("base64"),
				SHA:      gh.Ptr("abc123"),
			},
			nil,
			&gh.Response{},
			nil,
		)

	c := &client{repositories: repoSvc, org: "org-name"}

	content, sha, err := c.GetFileContent(ctx, "repo-name", ".github/workflows/test.yml", "main")

	assert.NoError(t, err)
	assert.Equal(t, fileContent, content)
	assert.Equal(t, "abc123", sha)
}

func TestGetFileContent_NotFound(t *testing.T) {
	ctx := context.Background()
	repoSvc := github.NewMockRepositoriesAdapter(t)

	repoSvc.
		EXPECT().
		GetContents(mock.Anything, "org-name", "repo-name", ".github/workflows/test.yml", mock.Anything).
		Once().
		Return(nil, nil, nil, errors.New("not found"))

	c := &client{repositories: repoSvc, org: "org-name"}

	content, sha, err := c.GetFileContent(ctx, "repo-name", ".github/workflows/test.yml", "main")

	assert.Error(t, err)
	assert.Empty(t, content)
	assert.Empty(t, sha)
}

func TestCreateOrUpdateFile_Create(t *testing.T) {
	ctx := context.Background()
	repoSvc := github.NewMockRepositoriesAdapter(t)

	repoSvc.
		EXPECT().
		CreateFile(mock.Anything, "org-name", "repo-name", ".github/workflows/test.yml",
			mock.MatchedBy(func(opts *gh.RepositoryContentFileOptions) bool {
				return opts.GetMessage() == "Add workflow" &&
					opts.GetBranch() == "feature-branch" &&
					string(opts.Content) == "workflow content" &&
					opts.SHA == nil
			}),
		).
		Once().
		Return(&gh.RepositoryContentResponse{}, &gh.Response{}, nil)

	c := &client{repositories: repoSvc, org: "org-name"}

	err := c.CreateOrUpdateFile(ctx, "repo-name", ".github/workflows/test.yml", "feature-branch", "Add workflow", "workflow content", nil)

	assert.NoError(t, err)
}

func TestCreateOrUpdateFile_Update(t *testing.T) {
	ctx := context.Background()
	repoSvc := github.NewMockRepositoriesAdapter(t)

	fileSHA := "existing-sha"

	repoSvc.
		EXPECT().
		UpdateFile(mock.Anything, "org-name", "repo-name", ".github/workflows/test.yml",
			mock.MatchedBy(func(opts *gh.RepositoryContentFileOptions) bool {
				return opts.GetMessage() == "Update workflow" &&
					opts.GetBranch() == "feature-branch" &&
					string(opts.Content) == "updated content" &&
					opts.GetSHA() == "existing-sha"
			}),
		).
		Once().
		Return(&gh.RepositoryContentResponse{}, &gh.Response{}, nil)

	c := &client{repositories: repoSvc, org: "org-name"}

	err := c.CreateOrUpdateFile(ctx, "repo-name", ".github/workflows/test.yml", "feature-branch", "Update workflow", "updated content", &fileSHA)

	assert.NoError(t, err)
}

func TestCreateOrUpdateFile_CreateError(t *testing.T) {
	ctx := context.Background()
	repoSvc := github.NewMockRepositoriesAdapter(t)

	repoSvc.
		EXPECT().
		CreateFile(mock.Anything, "org-name", "repo-name", mock.Anything, mock.Anything).
		Once().
		Return(nil, nil, errors.New("permission denied"))

	c := &client{repositories: repoSvc, org: "org-name"}

	err := c.CreateOrUpdateFile(ctx, "repo-name", ".github/workflows/test.yml", "feature-branch", "Add workflow", "content", nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestCreateOrUpdateFile_UpdateError(t *testing.T) {
	ctx := context.Background()
	repoSvc := github.NewMockRepositoriesAdapter(t)

	fileSHA := "existing-sha"

	repoSvc.
		EXPECT().
		UpdateFile(mock.Anything, "org-name", "repo-name", mock.Anything, mock.Anything).
		Once().
		Return(nil, nil, errors.New("conflict"))

	c := &client{repositories: repoSvc, org: "org-name"}

	err := c.CreateOrUpdateFile(ctx, "repo-name", ".github/workflows/test.yml", "feature-branch", "Update workflow", "content", &fileSHA)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "conflict")
}
