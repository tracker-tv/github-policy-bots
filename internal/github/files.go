package github

import (
	"context"

	gh "github.com/google/go-github/v80/github"
)

func (c *client) GetFileContent(ctx context.Context, repo, path, ref string) (string, string, error) {
	opts := &gh.RepositoryContentGetOptions{Ref: ref}
	content, _, _, err := c.repositories.GetContents(ctx, c.org, repo, path, opts)
	if err != nil {
		return "", "", err
	}
	decoded, err := content.GetContent()
	if err != nil {
		return "", "", err
	}
	return decoded, content.GetSHA(), nil
}

func (c *client) CreateOrUpdateFile(ctx context.Context, repo, path, branch, message, content string, fileSHA *string) error {
	opts := &gh.RepositoryContentFileOptions{
		Message: gh.Ptr(message),
		Content: []byte(content),
		Branch:  gh.Ptr(branch),
	}
	if fileSHA != nil {
		opts.SHA = fileSHA
	}

	if fileSHA == nil {
		_, _, err := c.repositories.CreateFile(ctx, c.org, repo, path, opts)
		return err
	}
	_, _, err := c.repositories.UpdateFile(ctx, c.org, repo, path, opts)
	return err
}
