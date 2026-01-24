package github

import (
	"context"

	gh "github.com/google/go-github/v80/github"
)

func (c *client) ListPullRequests(ctx context.Context, repo string, opts *gh.PullRequestListOptions) ([]*gh.PullRequest, error) {
	prs, _, err := c.pullRequests.List(ctx, c.org, repo, opts)
	return prs, err
}

func (c *client) CreatePullRequest(ctx context.Context, repo, title, body, head, base string) (*gh.PullRequest, error) {
	pr := &gh.NewPullRequest{
		Title: gh.Ptr(title),
		Body:  gh.Ptr(body),
		Head:  gh.Ptr(head),
		Base:  gh.Ptr(base),
	}
	created, _, err := c.pullRequests.Create(ctx, c.org, repo, pr)
	return created, err
}

func (c *client) FindPullRequestByBranch(ctx context.Context, repo, branchName string) (*gh.PullRequest, error) {
	opts := &gh.PullRequestListOptions{
		Head:  c.org + ":" + branchName,
		State: "open",
	}
	prs, err := c.ListPullRequests(ctx, repo, opts)
	if err != nil {
		return nil, err
	}
	if len(prs) > 0 {
		return prs[0], nil
	}
	return nil, nil
}
