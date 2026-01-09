package github

import (
	"context"
	"errors"
	"testing"
	"time"

	gh "github.com/google/go-github/v80/github"
)

type mockRepositoriesService struct{ call int }
type rateLimitMockRepoService struct{ call int }
type blockingMockRepoService struct{}

func (m *mockRepositoriesService) ListByOrg(ctx context.Context, org string, opts *gh.RepositoryListByOrgOptions) ([]*gh.Repository, *gh.Response, error) {
	if m.call == 0 {
		m.call++
		return nil, nil, &gh.RateLimitError{
			Rate: gh.Rate{Reset: gh.Timestamp{Time: time.Now().Add(-1 * time.Second)}},
		}
	}
	if opts.Page == 0 {
		return []*gh.Repository{
			{ID: gh.Ptr(int64(1)), Name: gh.Ptr("repo-1")},
			{ID: gh.Ptr(int64(2)), Name: gh.Ptr("repo-2")},
		}, &gh.Response{NextPage: 2}, nil
	}
	if opts.Page == 2 {
		return []*gh.Repository{
			{ID: gh.Ptr(int64(3)), Name: gh.Ptr("repo-3")},
		}, &gh.Response{NextPage: 0}, nil
	}
	return nil, nil, errors.New("unexpected call")
}

func (m *rateLimitMockRepoService) ListByOrg(ctx context.Context, org string, opts *gh.RepositoryListByOrgOptions) ([]*gh.Repository, *gh.Response, error) {
	if m.call == 0 {
		m.call++
		return nil, nil, &gh.RateLimitError{
			Rate: gh.Rate{Reset: gh.Timestamp{Time: time.Now().Add(-1 * time.Second)}},
		}
	}
	return []*gh.Repository{
		{ID: gh.Ptr(int64(42)), Name: gh.Ptr("rate-limit-ok")},
	}, &gh.Response{NextPage: 0}, nil
}

func (m *blockingMockRepoService) ListByOrg(ctx context.Context, org string, opts *gh.RepositoryListByOrgOptions) ([]*gh.Repository, *gh.Response, error) {
	return nil, nil, &gh.RateLimitError{
		Rate: gh.Rate{Reset: gh.Timestamp{Time: time.Now().Add(10 * time.Second)}},
	}
}

func TestListAllRepos(t *testing.T) {
	tests := []struct {
		name         string
		repoService  RepositoriesService
		wantRepos    []string
		wantCall     int
		expectErr    bool
		contextDelay time.Duration
	}{
		{
			name:        "pagination and retry",
			repoService: &mockRepositoriesService{},
			wantRepos:   []string{"repo-1", "repo-2", "repo-3"},
			wantCall:    1,
		},
		{
			name:        "rate limit retry",
			repoService: &rateLimitMockRepoService{},
			wantRepos:   []string{"rate-limit-ok"},
			wantCall:    1,
		},
		{
			name:         "context timeout",
			repoService:  &blockingMockRepoService{},
			wantRepos:    []string{},
			expectErr:    true,
			contextDelay: 1 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx context.Context
			var cancel context.CancelFunc
			if tt.contextDelay > 0 {
				ctx, cancel = context.WithTimeout(context.Background(), tt.contextDelay)
				defer cancel()
			} else {
				ctx = context.Background()
			}

			client := &Client{repositories: tt.repoService}
			start := time.Now()
			repos, err := client.ListAllRepos(ctx)
			elapsed := time.Since(start)

			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("expected context deadline exceeded, got %v", err)
				}
				if len(repos) != 0 {
					t.Fatalf("expected 0 repos, got %d", len(repos))
				}
				if elapsed > 50*time.Millisecond {
					t.Fatalf("context cancellation too slow: %v", elapsed)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(repos) != len(tt.wantRepos) {
				t.Fatalf("expected %d repos, got %d", len(tt.wantRepos), len(repos))
			}

			for i, r := range repos {
				if r.GetName() != tt.wantRepos[i] {
					t.Errorf("repo %d: expected %q, got %q", i, tt.wantRepos[i], r.GetName())
				}
			}

			if m, ok := tt.repoService.(*mockRepositoriesService); ok && m.call != tt.wantCall {
				t.Errorf("expected call %d, got %d", tt.wantCall, m.call)
			}
			if m, ok := tt.repoService.(*rateLimitMockRepoService); ok && m.call != tt.wantCall {
				t.Errorf("expected call %d, got %d", tt.wantCall, m.call)
			}
		})
	}
}
