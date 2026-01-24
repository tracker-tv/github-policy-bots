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

func TestListPullRequests_Success(t *testing.T) {
	ctx := context.Background()
	prSvc := github.NewMockPullRequestsAdapter(t)

	opts := &gh.PullRequestListOptions{State: "open"}

	prSvc.
		EXPECT().
		List(mock.Anything, "org-name", "repo-name", opts).
		Once().
		Return(
			[]*gh.PullRequest{
				{Number: gh.Ptr(1), Title: gh.Ptr("PR 1")},
				{Number: gh.Ptr(2), Title: gh.Ptr("PR 2")},
			},
			&gh.Response{},
			nil,
		)

	c := &client{pullRequests: prSvc, org: "org-name"}

	prs, err := c.ListPullRequests(ctx, "repo-name", opts)

	assert.NoError(t, err)
	assert.Len(t, prs, 2)
	assert.Equal(t, 1, prs[0].GetNumber())
	assert.Equal(t, 2, prs[1].GetNumber())
}

func TestListPullRequests_Error(t *testing.T) {
	ctx := context.Background()
	prSvc := github.NewMockPullRequestsAdapter(t)

	prSvc.
		EXPECT().
		List(mock.Anything, "org-name", "repo-name", mock.Anything).
		Once().
		Return(nil, nil, errors.New("API error"))

	c := &client{pullRequests: prSvc, org: "org-name"}

	prs, err := c.ListPullRequests(ctx, "repo-name", nil)

	assert.Error(t, err)
	assert.Nil(t, prs)
	assert.Contains(t, err.Error(), "API error")
}

func TestCreatePullRequest_Success(t *testing.T) {
	ctx := context.Background()
	prSvc := github.NewMockPullRequestsAdapter(t)

	prSvc.
		EXPECT().
		Create(mock.Anything, "org-name", "repo-name",
			mock.MatchedBy(func(pr *gh.NewPullRequest) bool {
				return pr.GetTitle() == "Test PR" &&
					pr.GetBody() == "PR body" &&
					pr.GetHead() == "feature-branch" &&
					pr.GetBase() == "main"
			}),
		).
		Once().
		Return(
			&gh.PullRequest{
				Number:  gh.Ptr(42),
				Title:   gh.Ptr("Test PR"),
				HTMLURL: gh.Ptr("https://github.com/org-name/repo-name/pull/42"),
			},
			&gh.Response{},
			nil,
		)

	c := &client{pullRequests: prSvc, org: "org-name"}

	pr, err := c.CreatePullRequest(ctx, "repo-name", "Test PR", "PR body", "feature-branch", "main")

	assert.NoError(t, err)
	assert.NotNil(t, pr)
	assert.Equal(t, 42, pr.GetNumber())
	assert.Equal(t, "https://github.com/org-name/repo-name/pull/42", pr.GetHTMLURL())
}

func TestCreatePullRequest_Error(t *testing.T) {
	ctx := context.Background()
	prSvc := github.NewMockPullRequestsAdapter(t)

	prSvc.
		EXPECT().
		Create(mock.Anything, "org-name", "repo-name", mock.Anything).
		Once().
		Return(nil, nil, errors.New("PR already exists"))

	c := &client{pullRequests: prSvc, org: "org-name"}

	pr, err := c.CreatePullRequest(ctx, "repo-name", "Test PR", "PR body", "feature-branch", "main")

	assert.Error(t, err)
	assert.Nil(t, pr)
	assert.Contains(t, err.Error(), "PR already exists")
}

func TestFindPullRequestByBranch_Found(t *testing.T) {
	ctx := context.Background()
	prSvc := github.NewMockPullRequestsAdapter(t)

	prSvc.
		EXPECT().
		List(mock.Anything, "org-name", "repo-name",
			mock.MatchedBy(func(opts *gh.PullRequestListOptions) bool {
				return opts.Head == "org-name:chore/dockerfile" && opts.State == "open"
			}),
		).
		Once().
		Return(
			[]*gh.PullRequest{
				{Number: gh.Ptr(1), Title: gh.Ptr("Policy PR")},
			},
			&gh.Response{},
			nil,
		)

	c := &client{pullRequests: prSvc, org: "org-name"}

	pr, err := c.FindPullRequestByBranch(ctx, "repo-name", "chore/dockerfile")

	assert.NoError(t, err)
	assert.NotNil(t, pr)
	assert.Equal(t, 1, pr.GetNumber())
}

func TestFindPullRequestByBranch_NotFound(t *testing.T) {
	ctx := context.Background()
	prSvc := github.NewMockPullRequestsAdapter(t)

	prSvc.
		EXPECT().
		List(mock.Anything, "org-name", "repo-name", mock.Anything).
		Once().
		Return([]*gh.PullRequest{}, &gh.Response{}, nil)

	c := &client{pullRequests: prSvc, org: "org-name"}

	pr, err := c.FindPullRequestByBranch(ctx, "repo-name", "chore/dockerfile")

	assert.NoError(t, err)
	assert.Nil(t, pr)
}

func TestFindPullRequestByBranch_Error(t *testing.T) {
	ctx := context.Background()
	prSvc := github.NewMockPullRequestsAdapter(t)

	prSvc.
		EXPECT().
		List(mock.Anything, "org-name", "repo-name", mock.Anything).
		Once().
		Return(nil, nil, errors.New("API error"))

	c := &client{pullRequests: prSvc, org: "org-name"}

	pr, err := c.FindPullRequestByBranch(ctx, "repo-name", "chore/dockerfile")

	assert.Error(t, err)
	assert.Nil(t, pr)
}
