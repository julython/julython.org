package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const githubAPIBase = "https://api.github.com"

type GitHubService struct {
	token  string
	client *http.Client
}

type GitHubRepo struct {
	ID            int               `json:"id"`
	Name          string            `json:"name"`
	FullName      string            `json:"full_name"`
	Owner         GitHubOwner       `json:"owner"`
	Private       bool              `json:"private"`
	HTMLURL       string            `json:"html_url"`
	Description   *string           `json:"description"`
	DefaultBranch string            `json:"default_branch"`
	HooksURL      string            `json:"hooks_url"`
	Forked        bool              `json:"fork"`
	Permissions   GitHubPermissions `json:"permissions"`
	Webhooks      []GitHubWebhook   `json:"-"` // Populated separately
}

type GitHubOwner struct {
	Login string `json:"login"`
}

type GitHubPermissions struct {
	Admin bool `json:"admin"`
	Push  bool `json:"push"`
	Pull  bool `json:"pull"`
}

type GitHubWebhook struct {
	ID     int                 `json:"id"`
	Name   string              `json:"name"`
	Active bool                `json:"active"`
	Events []string            `json:"events"`
	Config GitHubWebhookConfig `json:"config"`
}

type GitHubWebhookConfig struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	InsecureSSL string `json:"insecure_ssl"`
}

// RepoWithWebhook augments GitHubRepo with the Julython webhook state.
// Populated by the profile handler after calling ListRepos(includeWebhooks=true).
type RepoWithWebhook struct {
	GitHubRepo
	HasWebhook bool
	WebhookID  int
}

func NewGitHubService(accessToken string) *GitHubService {
	return &GitHubService{
		token: accessToken,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *GitHubService) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	return s.client.Do(req)
}

// ListRepos lists repositories the user has admin access to.
func (s *GitHubService) ListRepos(ctx context.Context, includeWebhooks bool, perPage, page int) ([]GitHubRepo, error) {
	url := fmt.Sprintf("%s/user/repos?per_page=%d&page=%d&sort=updated&direction=desc&affiliation=owner,organization_member",
		githubAPIBase, perPage, page)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := s.do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github api error: %d %s", resp.StatusCode, string(body))
	}

	var repos []GitHubRepo
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Filter to only repos with admin access
	filtered := make([]GitHubRepo, 0, len(repos))
	for _, repo := range repos {
		if repo.Permissions.Admin {
			filtered = append(filtered, repo)
		}
	}

	if includeWebhooks {
		for i := range filtered {
			webhooks, err := s.GetRepoWebhooks(ctx, filtered[i].Owner.Login, filtered[i].Name)
			if err == nil {
				filtered[i].Webhooks = webhooks
			}
			// Ignore errors - user might not have permission
		}
	}

	return filtered, nil
}

// GetRepoWebhooks fetches webhooks for a specific repository.
func (s *GitHubService) GetRepoWebhooks(ctx context.Context, owner, repo string) ([]GitHubWebhook, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/hooks", githubAPIBase, owner, repo)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := s.do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // No permission
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api error: %d", resp.StatusCode)
	}

	var webhooks []GitHubWebhook
	if err := json.NewDecoder(resp.Body).Decode(&webhooks); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return webhooks, nil
}

// CreateWebhook creates a new webhook on a repository.
func (s *GitHubService) CreateWebhook(ctx context.Context, owner, repo, webhookURL string, events []string) (GitHubWebhook, error) {
	if events == nil {
		events = []string{"push"}
	}

	payload := map[string]interface{}{
		"name":   "web",
		"active": true,
		"events": events,
		"config": map[string]string{
			"url":          webhookURL,
			"content_type": "json",
			"insecure_ssl": "0",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return GitHubWebhook{}, fmt.Errorf("marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/hooks", githubAPIBase, owner, repo)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return GitHubWebhook{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.do(req)
	if err != nil {
		return GitHubWebhook{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return GitHubWebhook{}, fmt.Errorf("github api error: %d %s", resp.StatusCode, string(body))
	}

	var webhook GitHubWebhook
	if err := json.NewDecoder(resp.Body).Decode(&webhook); err != nil {
		return GitHubWebhook{}, fmt.Errorf("decode response: %w", err)
	}

	return webhook, nil
}

// DeleteWebhook removes a webhook from a repository.
func (s *GitHubService) DeleteWebhook(ctx context.Context, owner, repo string, hookID int) error {
	url := fmt.Sprintf("%s/repos/%s/%s/hooks/%d", githubAPIBase, owner, repo, hookID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := s.do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("github api error: %d", resp.StatusCode)
	}

	return nil
}

// PingWebhook triggers a ping event for a webhook.
func (s *GitHubService) PingWebhook(ctx context.Context, owner, repo string, hookID int) error {
	url := fmt.Sprintf("%s/repos/%s/%s/hooks/%d/pings", githubAPIBase, owner, repo, hookID)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := s.do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("github api error: %d", resp.StatusCode)
	}

	return nil
}
