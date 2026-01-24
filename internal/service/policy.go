package service

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/tracker-tv/github-policy-bots/internal/github"
	"github.com/tracker-tv/github-policy-bots/models"
)

type PolicyService interface {
	Ensure(ctx context.Context, repo models.Repository, repoFiles []string) ([]models.PolicyDeviation, error)
}

type policyService struct {
	workflows  []models.PolicyWorkflow
	gh         github.Client
	httpClient *http.Client
}

func NewPolicyService(workflows []models.PolicyWorkflow, gh github.Client) PolicyService {
	return &policyService{
		workflows:  workflows,
		gh:         gh,
		httpClient: http.DefaultClient,
	}
}

func (s *policyService) Ensure(ctx context.Context, repo models.Repository, repoFiles []string) ([]models.PolicyDeviation, error) {
	var deviations []models.PolicyDeviation

	for _, policy := range s.workflows {
		matched, err := s.matchesPolicy(repoFiles, policy.MatchFile)
		if err != nil {
			return nil, fmt.Errorf("matching policy %s: %w", policy.Name, err)
		}

		if !matched {
			continue
		}

		targetPath := fmt.Sprintf(".github/workflows/%s.yml", policy.Name)

		content, _, resp, err := s.gh.GetContentsRaw(ctx, repo.Name, targetPath)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				deviations = append(deviations, models.PolicyDeviation{
					Repository:     repo,
					Policy:         policy,
					Action:         models.PolicyActionCreate,
					TargetPath:     targetPath,
					ExpectedSource: policy.Source,
					CurrentContent: "",
				})
				continue
			}
			return nil, fmt.Errorf("getting workflow %s: %w", targetPath, err)
		}

		currentContent, err := content.GetContent()
		if err != nil {
			return nil, fmt.Errorf("decoding workflow content %s: %w", targetPath, err)
		}

		expectedContent, err := s.fetchExpectedContent(ctx, policy.Source)
		if err != nil {
			return nil, fmt.Errorf("fetching expected content for %s: %w", policy.Name, err)
		}

		wrappedExpectedContent := wrapContent(expectedContent, policy.Name)
		if currentContent != wrappedExpectedContent {
			deviations = append(deviations, models.PolicyDeviation{
				Repository:     repo,
				Policy:         policy,
				Action:         models.PolicyActionUpdate,
				TargetPath:     targetPath,
				ExpectedSource: policy.Source,
				CurrentContent: currentContent,
			})
		}
	}

	return deviations, nil
}

func (s *policyService) matchesPolicy(files []string, pattern string) (bool, error) {
	for _, file := range files {
		matched, err := doublestar.Match(pattern, file)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func (s *policyService) fetchExpectedContent(ctx context.Context, sourceURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
