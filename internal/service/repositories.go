package service

import (
	"context"

	"github.com/tracker-tv/github-policy-bots/internal/github"
	"github.com/tracker-tv/github-policy-bots/models"
)

type RepositoryService interface {
	ListAll(ctx context.Context) ([]models.Repository, error)
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
