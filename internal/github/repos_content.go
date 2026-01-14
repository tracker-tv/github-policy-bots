package github

import (
	"context"

	gh "github.com/google/go-github/v80/github"
)

func (c *client) GetContentsRaw(ctx context.Context, repo, path string) (*gh.RepositoryContent, []*gh.RepositoryContent, *gh.Response, error) {
	opts := &gh.RepositoryContentGetOptions{}
	return c.repositories.GetContents(ctx, c.org, repo, path, opts)
}
