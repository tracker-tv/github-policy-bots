package github

import (
	"context"
	"net/http"

	gh "github.com/google/go-github/v80/github"
)

type Client interface {
	// Repository operations
	ListAllRepos(ctx context.Context) ([]*gh.Repository, error)
	GetContentsRaw(ctx context.Context, repo, path string) (*gh.RepositoryContent, []*gh.RepositoryContent, *gh.Response, error)
	GetTree(ctx context.Context, repo, sha string, recursive bool) (*gh.Tree, *gh.Response, error)

	// Branch operations
	GetBranch(ctx context.Context, repo, branch string) (*gh.Reference, error)
	CreateBranch(ctx context.Context, repo, branchName, baseSHA string) error

	// File operations
	GetFileContent(ctx context.Context, repo, path, ref string) (content string, sha string, err error)
	CreateOrUpdateFile(ctx context.Context, repo, path, branch, message, content string, fileSHA *string) error

	// Pull request operations
	ListPullRequests(ctx context.Context, repo string, opts *gh.PullRequestListOptions) ([]*gh.PullRequest, error)
	CreatePullRequest(ctx context.Context, repo, title, body, head, base string) (*gh.PullRequest, error)
	FindPullRequestByBranch(ctx context.Context, repo, branchName string) (*gh.PullRequest, error)
}

type RepositoriesAdapter interface {
	ListByOrg(ctx context.Context, org string, opts *gh.RepositoryListByOrgOptions) ([]*gh.Repository, *gh.Response, error)
	GetContents(ctx context.Context, owner, repo, path string, opts *gh.RepositoryContentGetOptions) (*gh.RepositoryContent, []*gh.RepositoryContent, *gh.Response, error)
	CreateFile(ctx context.Context, owner, repo, path string, opts *gh.RepositoryContentFileOptions) (*gh.RepositoryContentResponse, *gh.Response, error)
	UpdateFile(ctx context.Context, owner, repo, path string, opts *gh.RepositoryContentFileOptions) (*gh.RepositoryContentResponse, *gh.Response, error)
}

type GitAdapter interface {
	GetTree(ctx context.Context, owner, repo, sha string, recursive bool) (*gh.Tree, *gh.Response, error)
}

type ReferencesAdapter interface {
	GetRef(ctx context.Context, owner, repo, ref string) (*gh.Reference, *gh.Response, error)
	CreateRef(ctx context.Context, owner, repo string, ref gh.CreateRef) (*gh.Reference, *gh.Response, error)
}

type PullRequestsAdapter interface {
	List(ctx context.Context, owner, repo string, opts *gh.PullRequestListOptions) ([]*gh.PullRequest, *gh.Response, error)
	Create(ctx context.Context, owner, repo string, pull *gh.NewPullRequest) (*gh.PullRequest, *gh.Response, error)
}

type client struct {
	github       *gh.Client
	repositories RepositoriesAdapter
	git          GitAdapter
	references   ReferencesAdapter
	pullRequests PullRequestsAdapter
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
		references:   c.Git,
		pullRequests: c.PullRequests,
		org:          org,
	}
}
