package github

import (
	"context"
	"errors"
	"fmt"
	"time"

	gh "github.com/google/go-github/v80/github"
)

func (c *client) ListAllRepos(ctx context.Context) ([]*gh.Repository, error) {
	var allRepos []*gh.Repository

	opts := &gh.RepositoryListByOrgOptions{
		Sort:        "full_name",
		ListOptions: gh.ListOptions{PerPage: 100},
	}

	for {
		repos, resp, err := c.listReposWithRetry(ctx, opts)
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

func (c *client) listReposWithRetry(ctx context.Context, opts *gh.RepositoryListByOrgOptions) ([]*gh.Repository, *gh.Response, error) {
	const maxRetries = 5
	const baseDelay = time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		repos, resp, err := c.repositories.ListByOrg(ctx, c.org, opts)
		if err == nil {
			return repos, resp, nil
		}

		var rateErr *gh.RateLimitError
		if !errors.As(err, &rateErr) {
			return nil, nil, err
		}

		if attempt == maxRetries {
			return nil, nil, fmt.Errorf("max retries reached: %w", err)
		}

		wait := rateErr.Rate.Reset.Sub(time.Now())
		if wait < 0 {
			wait = baseDelay * time.Duration(1<<attempt)
		}

		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		}
	}

	return nil, nil, fmt.Errorf("unreachable")
}
