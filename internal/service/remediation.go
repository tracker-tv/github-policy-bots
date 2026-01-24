package service

import (
	"context"
	"fmt"
	"io"
	"net/http"

	gh "github.com/google/go-github/v80/github"
	"github.com/tracker-tv/github-policy-bots/internal/github"
	"github.com/tracker-tv/github-policy-bots/models"
)

type RemediationResult struct {
	Drift  models.PolicyDeviation
	Action string // "created", "updated", "skipped"
	PRURL  string
	Error  error
}

type RemediationService interface {
	Remediate(ctx context.Context, drift models.PolicyDeviation) (*RemediationResult, error)
}

type remediationService struct {
	gh         github.Client
	httpClient *http.Client
}

func NewRemediationService(gh github.Client) RemediationService {
	return &remediationService{
		gh:         gh,
		httpClient: http.DefaultClient,
	}
}

func (s *remediationService) Remediate(ctx context.Context, drift models.PolicyDeviation) (*RemediationResult, error) {
	branchName := s.branchName(drift)

	// 1. Fetch expected content from source
	expectedContent, err := s.fetchExpectedContent(ctx, drift.ExpectedSource)
	if err != nil {
		return nil, fmt.Errorf("fetching expected content: %w", err)
	}

	// 2. Check if PR already exists for this branch
	existingPR, err := s.gh.FindPullRequestByBranch(ctx, drift.Repository.Name, branchName)
	if err != nil {
		return nil, fmt.Errorf("finding existing PR: %w", err)
	}

	if existingPR != nil {
		// PR exists - check if content needs update
		return s.handleExistingPR(ctx, drift, branchName, expectedContent, existingPR)
	}

	// 3. No existing PR - create new branch and PR
	return s.createNewPR(ctx, drift, branchName, expectedContent)
}

func (s *remediationService) branchName(drift models.PolicyDeviation) string {
	return fmt.Sprintf("chore/%s", drift.Policy.Name)
}

func (s *remediationService) handleExistingPR(ctx context.Context, drift models.PolicyDeviation, branchName, expectedContent string, pr *gh.PullRequest) (*RemediationResult, error) {
	// Get current content on the PR branch
	currentContent, fileSHA, err := s.gh.GetFileContent(ctx, drift.Repository.Name, drift.TargetPath, branchName)
	if err != nil {
		// File doesn't exist on branch yet - need to create it
		currentContent = ""
		fileSHA = ""
	}

	wrappedContent := wrapContent(expectedContent, drift.Policy.Name)

	if currentContent == wrappedContent {
		return &RemediationResult{
			Drift:  drift,
			Action: "skipped",
			PRURL:  pr.GetHTMLURL(),
		}, nil
	}

	// Content differs - update the file
	commitMsg := fmt.Sprintf("Update %s policy", drift.Policy.Name)
	var sha *string
	if fileSHA != "" {
		sha = &fileSHA
	}

	if err := s.gh.CreateOrUpdateFile(ctx, drift.Repository.Name, drift.TargetPath, branchName, commitMsg, wrappedContent, sha); err != nil {
		return nil, fmt.Errorf("updating file: %w", err)
	}

	return &RemediationResult{
		Drift:  drift,
		Action: "updated",
		PRURL:  pr.GetHTMLURL(),
	}, nil
}

func (s *remediationService) createNewPR(ctx context.Context, drift models.PolicyDeviation, branchName, expectedContent string) (*RemediationResult, error) {
	// 1. Get default branch SHA
	defaultBranch, err := s.gh.GetBranch(ctx, drift.Repository.Name, "main")
	if err != nil {
		// Try 'master' as fallback
		defaultBranch, err = s.gh.GetBranch(ctx, drift.Repository.Name, "master")
		if err != nil {
			return nil, fmt.Errorf("getting default branch: %w", err)
		}
	}

	if defaultBranch == nil || defaultBranch.GetObject() == nil {
		return nil, fmt.Errorf("default branch reference is nil for %s", drift.Repository.Name)
	}

	baseSHA := defaultBranch.GetObject().GetSHA()
	if baseSHA == "" {
		return nil, fmt.Errorf("default branch SHA is empty for %s", drift.Repository.Name)
	}

	// 2. Create new branch (ignore error if branch already exists)
	if err := s.gh.CreateBranch(ctx, drift.Repository.Name, branchName, baseSHA); err != nil {
		// Check if branch already exists by trying to get it
		_, getErr := s.gh.GetBranch(ctx, drift.Repository.Name, branchName)
		if getErr != nil {
			// Branch doesn't exist and creation failed
			return nil, fmt.Errorf("creating branch %s: %w", branchName, err)
		}
		// Branch already exists, continue
	}

	// Verify branch exists before proceeding
	createdBranch, err := s.gh.GetBranch(ctx, drift.Repository.Name, branchName)
	if err != nil {
		return nil, fmt.Errorf("verifying branch %s exists: %w", branchName, err)
	}
	if createdBranch == nil {
		return nil, fmt.Errorf("branch %s was not created", branchName)
	}

	// 3. Create/update the file on the new branch
	commitMsg := fmt.Sprintf("chore(gha): %s %s workflow", actionVerb(drift.Action), drift.Policy.Name)

	// Get existing file SHA if updating
	var fileSHA *string
	if drift.Action == models.PolicyActionUpdate {
		_, sha, err := s.gh.GetFileContent(ctx, drift.Repository.Name, drift.TargetPath, "HEAD")
		if err == nil {
			fileSHA = &sha
		}
	}

	wrappedContent := wrapContent(expectedContent, drift.Policy.Name)
	if err := s.gh.CreateOrUpdateFile(ctx, drift.Repository.Name, drift.TargetPath, branchName, commitMsg, wrappedContent, fileSHA); err != nil {
		return nil, fmt.Errorf("creating file %s on branch %s: %w", drift.TargetPath, branchName, err)
	}

	// 4. Create PR
	prTitle := fmt.Sprintf("chore(gha): %s %s workflow", actionVerb(drift.Action), drift.Policy.Name)
	prBody := s.buildPRBody(drift)

	pr, err := s.gh.CreatePullRequest(ctx, drift.Repository.Name, prTitle, prBody, branchName, "main")
	if err != nil {
		return nil, fmt.Errorf("creating PR: %w", err)
	}

	return &RemediationResult{
		Drift:  drift,
		Action: "created",
		PRURL:  pr.GetHTMLURL(),
	}, nil
}

func (s *remediationService) buildPRBody(drift models.PolicyDeviation) string {
	return fmt.Sprintf(`## Policy Bot Automated PR

This PR was automatically created by the Policy Bot to ensure compliance.

**Policy:** %s
**Action:** %s
**Target File:** %s

---
*This is an automated PR. Please review before merging.*
`, drift.Policy.Name, drift.Action, drift.TargetPath)
}

func actionVerb(action models.PolicyAction) string {
	if action == models.PolicyActionCreate {
		return "add"
	}
	return "update"
}

func wrapContent(content, policyName string) string {
	header := fmt.Sprintf(`# DO NOT EDIT: BEGIN
# This snippet has been inserted automatically by tracker-tv-bot, do not edit!
# If changes are needed, update the action %s in
# https://github.com/tracker-tv/github-actions-ttv.
`, policyName)
	footer := "# DO NOT EDIT: END\n"
	return header + content + footer
}

func (s *remediationService) fetchExpectedContent(ctx context.Context, sourceURL string) (string, error) {
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
