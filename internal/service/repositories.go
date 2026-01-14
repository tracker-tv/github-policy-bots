package service

import (
	"context"

	"github.com/tracker-tv/github-policy-bots/internal/github"
	"github.com/tracker-tv/github-policy-bots/models"
)

type RepositoryService interface {
	ListAll(ctx context.Context) ([]models.Repository, error)
	ListFiles(ctx context.Context, repoName string) ([]string, error)
}

type repositoriesService struct {
	gh github.Client
}

func NewRepositoriesService(ghClient github.Client) RepositoryService {
	return &repositoriesService{gh: ghClient}
}

func (s *repositoriesService) ListAll(ctx context.Context) ([]models.Repository, error) {
	repos, err := s.gh.ListAllRepos(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]models.Repository, 0, len(repos))

	for _, repo := range repos {
		if repo == nil {
			continue
		}

		result = append(result, models.Repository{
			Name:     repo.GetName(),
			FullName: repo.GetFullName(),
			Private:  repo.GetPrivate(),
			Archived: repo.GetArchived(),
		})
	}

	return result, nil
}

func (s *repositoriesService) ListFiles(ctx context.Context, repoName string) ([]string, error) {
	tree, _, err := s.gh.GetTree(ctx, repoName, "HEAD", true)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0, len(tree.Entries))
	for _, entry := range tree.Entries {
		if entry.GetType() == "blob" {
			files = append(files, entry.GetPath())
		}
	}
	return files, nil
}
