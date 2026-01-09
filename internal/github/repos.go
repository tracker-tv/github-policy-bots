package github

import (
	"context"
	"errors"
	"fmt"
	"time"

	gh "github.com/google/go-github/v80/github"
)

func (c *Client) ListAllRepos(ctx context.Context) ([]*gh.Repository, error) {
	var allRepos []*gh.Repository
	opts := &gh.RepositoryListByOrgOptions{
		ListOptions: gh.ListOptions{
			PerPage: 100,
		},
	}

	for {
		repos, resp, err := c.listAllReposWithRetry(ctx, opts)
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

func (c *Client) listAllReposWithRetry(ctx context.Context, opts *gh.RepositoryListByOrgOptions) ([]*gh.Repository, *gh.Response, error) {
	maxRetries := 5
	baseDelay := 1 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		repos, resp, err := c.repositories.ListByOrg(ctx, "tracker-tv", opts)

		if err == nil {
			return repos, resp, nil
		}

		var rateLimitErr *gh.RateLimitError
		ok := errors.As(err, &rateLimitErr)
		if !ok {
			return nil, nil, err
		}

		if attempt == maxRetries {
			return nil, nil, fmt.Errorf("max retries reached: %w", err)
		}

		waitDuration := rateLimitErr.Rate.Reset.Sub(time.Now())
		if waitDuration < 0 {
			waitDuration = baseDelay * time.Duration(1<<attempt)
		}

		select {
		case <-time.After(waitDuration):
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		}
	}

	return nil, nil, fmt.Errorf("unexpected retry loop exit")
}
