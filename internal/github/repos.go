package github

import (
	"context"

	gh "github.com/google/go-github/v80/github"
)

func (c *client) ListAllRepos(ctx context.Context) ([]*gh.Repository, error) {
	var allRepos []*gh.Repository

	opts := &gh.RepositoryListByOrgOptions{
		Sort:        "full_name",
		ListOptions: gh.ListOptions{PerPage: 100},
	}

	for {
		repos, resp, err := c.repositories.ListByOrg(ctx, c.org, opts)
		if err != nil {
			return nil, err
		}

		allRepos = append(allRepos, repos...)

		if resp == nil || resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allRepos, nil
}
