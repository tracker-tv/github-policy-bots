package github

import (
	"context"
	"net/http"

	gh "github.com/google/go-github/v80/github"
)

type RepositoriesService interface {
	ListByOrg(ctx context.Context, org string, opts *gh.RepositoryListByOrgOptions) ([]*gh.Repository, *gh.Response, error)
}

type Client struct {
	github       *gh.Client
	repositories RepositoriesService
}

type authTransport struct {
	token string
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(req)
}

func New(token string) *Client {
	var httpClient *http.Client
	if token != "" {
		httpClient = &http.Client{
			Transport: &authTransport{
				token: token,
			},
		}
	}
	client := gh.NewClient(httpClient)
	return &Client{
		github:       client,
		repositories: client.Repositories,
	}
}
