package github

import (
	"context"
	"net/http"

	gh "github.com/google/go-github/v80/github"
)

type Client interface {
	ListAllRepos(ctx context.Context) ([]*gh.Repository, error)
	GetContentsRaw(ctx context.Context, repo, path string) (*gh.RepositoryContent, []*gh.RepositoryContent, *gh.Response, error)
	GetTree(ctx context.Context, repo, sha string, recursive bool) (*gh.Tree, *gh.Response, error)
}

type RepositoriesAdapter interface {
	ListByOrg(ctx context.Context, org string, opts *gh.RepositoryListByOrgOptions) ([]*gh.Repository, *gh.Response, error)
	GetContents(ctx context.Context, owner, repo, path string, opts *gh.RepositoryContentGetOptions) (*gh.RepositoryContent, []*gh.RepositoryContent, *gh.Response, error)
}

type GitAdapter interface {
	GetTree(ctx context.Context, owner, repo, sha string, recursive bool) (*gh.Tree, *gh.Response, error)
}

type client struct {
	github       *gh.Client
	repositories RepositoriesAdapter
	git          GitAdapter
	org          string
}

type authTransport struct {
	token string
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(req)
}

func New(token, org string) Client {
	var httpClient *http.Client
	if token != "" {
		httpClient = &http.Client{
			Transport: &authTransport{
				token: token,
			},
		}
	}
	c := gh.NewClient(httpClient)

	return &client{
		github:       c,
		repositories: c.Repositories,
		git:          c.Git,
		org:          org,
	}
}
