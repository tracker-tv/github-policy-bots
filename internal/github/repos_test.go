package github

import (
	"context"
	"errors"
	"testing"
	"time"

	gh "github.com/google/go-github/v80/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	github "github.com/tracker-tv/github-policy-bots/internal/github/mocks"
)

func TestListAllRepos_PaginationAndRetry(t *testing.T) {
	ctx := context.Background()

	reposSvc := github.NewMockRepositoriesAdapter(t)

	// Page 1
	reposSvc.
		EXPECT().
		ListByOrg(mock.Anything, "org-name",
			mock.MatchedBy(func(o *gh.RepositoryListByOrgOptions) bool {
				return o.Page == 0
			}),
		).
		Once().
		Return(
			[]*gh.Repository{
				{ID: gh.Ptr(int64(1)), Name: gh.Ptr("repo-1")},
				{ID: gh.Ptr(int64(2)), Name: gh.Ptr("repo-2")},
			},
			&gh.Response{NextPage: 2},
			nil,
		)

	// Page 2
	reposSvc.
		EXPECT().
		ListByOrg(mock.Anything, "org-name",
			mock.MatchedBy(func(o *gh.RepositoryListByOrgOptions) bool {
				return o.Page == 2
			}),
		).
		Once().
		Return(
			[]*gh.Repository{
				{ID: gh.Ptr(int64(3)), Name: gh.Ptr("repo-3")},
			},
			&gh.Response{NextPage: 0},
			nil,
		)

	c := &client{repositories: reposSvc, org: "org-name"}

	repos, err := c.ListAllRepos(ctx)

	assert.NoError(t, err)
	assert.Len(t, repos, 3)
	assert.Equal(t, []string{"repo-1", "repo-2", "repo-3"}, []string{
		repos[0].GetName(),
		repos[1].GetName(),
		repos[2].GetName(),
	})
}

func TestListAllRepos_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	reposSvc := github.NewMockRepositoriesAdapter(t)

	reposSvc.
		EXPECT().
		ListByOrg(mock.Anything, "org-name", mock.Anything).
		RunAndReturn(func(ctx context.Context, _ string, _ *gh.RepositoryListByOrgOptions) ([]*gh.Repository, *gh.Response, error) {
			select {
			case <-time.After(100 * time.Millisecond):
				return []*gh.Repository{}, &gh.Response{}, nil
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			}
		})

	c := &client{repositories: reposSvc, org: "org-name"}

	start := time.Now()
	repos, err := c.ListAllRepos(ctx)
	elapsed := time.Since(start)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded))
	assert.Len(t, repos, 0)
	assert.Less(t, elapsed, 50*time.Millisecond)
}
