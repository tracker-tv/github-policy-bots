package github

import (
	"context"

	gh "github.com/google/go-github/v80/github"
)

func (c *client) GetBranch(ctx context.Context, repo, branch string) (*gh.Reference, error) {
	ref, _, err := c.references.GetRef(ctx, c.org, repo, "refs/heads/"+branch)
	return ref, err
}

func (c *client) CreateBranch(ctx context.Context, repo, branchName, baseSHA string) error {
	ref := gh.CreateRef{
		Ref: "refs/heads/" + branchName,
		SHA: baseSHA,
	}
	_, _, err := c.references.CreateRef(ctx, c.org, repo, ref)
	return err
}
