package github

import (
	"context"

	gh "github.com/google/go-github/v80/github"
)

func (c *client) GetTree(ctx context.Context, repo, sha string, recursive bool) (*gh.Tree, *gh.Response, error) {
	return c.git.GetTree(ctx, c.org, repo, sha, recursive)
}
