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

func TestGetBranch_Success(t *testing.T) {
	ctx := context.Background()
	refSvc := github.NewMockReferencesAdapter(t)

	refSvc.
		EXPECT().
		GetRef(mock.Anything, "org-name", "repo-name", "refs/heads/main").
		Once().
		Return(
			&gh.Reference{
				Ref: gh.Ptr("refs/heads/main"),
				Object: &gh.GitObject{
					SHA: gh.Ptr("abc123def456"),
				},
			},
			&gh.Response{},
			nil,
		)

	c := &client{references: refSvc, org: "org-name"}

	ref, err := c.GetBranch(ctx, "repo-name", "main")

	assert.NoError(t, err)
	assert.NotNil(t, ref)
	assert.Equal(t, "refs/heads/main", ref.GetRef())
	assert.Equal(t, "abc123def456", ref.GetObject().GetSHA())
}

func TestGetBranch_NotFound(t *testing.T) {
	ctx := context.Background()
	refSvc := github.NewMockReferencesAdapter(t)

	refSvc.
		EXPECT().
		GetRef(mock.Anything, "org-name", "repo-name", "refs/heads/nonexistent").
		Once().
		Return(nil, nil, errors.New("not found"))

	c := &client{references: refSvc, org: "org-name"}

	ref, err := c.GetBranch(ctx, "repo-name", "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, ref)
	assert.Contains(t, err.Error(), "not found")
}

func TestCreateBranch_Success(t *testing.T) {
	ctx := context.Background()
	refSvc := github.NewMockReferencesAdapter(t)

	refSvc.
		EXPECT().
		CreateRef(mock.Anything, "org-name", "repo-name",
			mock.MatchedBy(func(ref gh.CreateRef) bool {
				return ref.Ref == "refs/heads/chore/dockerfile" &&
					ref.SHA == "base-sha-123"
			}),
		).
		Once().
		Return(
			&gh.Reference{
				Ref: gh.Ptr("refs/heads/chore/dockerfile"),
				Object: &gh.GitObject{
					SHA: gh.Ptr("base-sha-123"),
				},
			},
			&gh.Response{},
			nil,
		)

	c := &client{references: refSvc, org: "org-name"}

	err := c.CreateBranch(ctx, "repo-name", "chore/dockerfile", "base-sha-123")

	assert.NoError(t, err)
}

func TestCreateBranch_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	refSvc := github.NewMockReferencesAdapter(t)

	refSvc.
		EXPECT().
		CreateRef(mock.Anything, "org-name", "repo-name", mock.Anything).
		Once().
		Return(nil, nil, errors.New("reference already exists"))

	c := &client{references: refSvc, org: "org-name"}

	err := c.CreateBranch(ctx, "repo-name", "chore/dockerfile", "base-sha-123")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reference already exists")
}

func TestCreateBranch_Error(t *testing.T) {
	ctx := context.Background()
	refSvc := github.NewMockReferencesAdapter(t)

	refSvc.
		EXPECT().
		CreateRef(mock.Anything, "org-name", "repo-name", mock.Anything).
		Once().
		Return(nil, nil, errors.New("permission denied"))

	c := &client{references: refSvc, org: "org-name"}

	err := c.CreateBranch(ctx, "repo-name", "feature-branch", "base-sha")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}
