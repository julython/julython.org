package services

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"crypto/rand"
)

var (
	ErrOAuthExchange       = errors.New("oauth token exchange failed")
	ErrOAuthUserFetch      = errors.New("failed to fetch oauth user")
	ErrTokenRevoke         = errors.New("failed to revoke token")
	ErrRefreshNotSupported = errors.New("token refresh not supported")
)

// OAuthTokens holds the tokens returned from an OAuth exchange.
type OAuthTokens struct {
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
}

// EmailAddress represents a user's email with verification status.
type EmailAddress struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// OAuthUser contains user info from an OAuth provider.
type OAuthUser struct {
	ID        string         `json:"id"`
	Provider  string         `json:"provider"`
	Username  string         `json:"username"`
	Name      string         `json:"name,omitempty"`
	AvatarURL string         `json:"avatar_url,omitempty"`
	Emails    []EmailAddress `json:"emails"`
	Data      map[string]any `json:"data,omitempty"` // Raw token data for storage
}

// Key returns the identifier key in "provider:id" format.
func (u OAuthUser) Key() string {
	return fmt.Sprintf("%s:%s", u.Provider, u.ID)
}

// OAuthProvider defines the interface for OAuth implementations.
type OAuthProvider interface {
	Provider() string
	AuthorizationURL(state string, pkceChallenge string) string
	ExchangeCode(ctx context.Context, code string, pkceVerifier string) (OAuthTokens, error)
	RefreshToken(ctx context.Context, refreshToken string) (OAuthTokens, error)
	GetUser(ctx context.Context, tokens OAuthTokens) (OAuthUser, error)
	RevokeToken(ctx context.Context, tokens OAuthTokens) error
}

// GitHubOAuth implements OAuth for GitHub.
type GitHubOAuth struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       string
	httpClient   *http.Client
}

func NewGitHubOAuth(clientID, clientSecret, redirectURI string) *GitHubOAuth {
	return &GitHubOAuth{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  redirectURI,
		Scopes:       "read:user user:email admin:repo_hook",
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (g *GitHubOAuth) Provider() string { return "github" }

func (g *GitHubOAuth) AuthorizationURL(state, _ string) string {
	params := url.Values{
		"client_id":     {g.ClientID},
		"redirect_uri":  {g.RedirectURI},
		"response_type": {"code"},
		"state":         {state},
		"scope":         {g.Scopes},
	}
	return "https://github.com/login/oauth/authorize?" + params.Encode()
}

func (g *GitHubOAuth) ExchangeCode(ctx context.Context, code, _ string) (OAuthTokens, error) {
	data := url.Values{
		"client_id":     {g.ClientID},
		"client_secret": {g.ClientSecret},
		"code":          {code},
		"redirect_uri":  {g.RedirectURI},
	}

	req, _ := http.NewRequestWithContext(ctx, "POST", "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return OAuthTokens{}, fmt.Errorf("%w: %v", ErrOAuthExchange, err)
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return OAuthTokens{}, fmt.Errorf("%w: decode: %v", ErrOAuthExchange, err)
	}
	if result.Error != "" {
		return OAuthTokens{}, fmt.Errorf("%w: %s", ErrOAuthExchange, result.ErrorDesc)
	}

	return OAuthTokens{AccessToken: result.AccessToken}, nil
}

func (g *GitHubOAuth) RefreshToken(_ context.Context, _ string) (OAuthTokens, error) {
	return OAuthTokens{}, ErrRefreshNotSupported // GitHub tokens don't expire
}

func (g *GitHubOAuth) GetUser(ctx context.Context, tokens OAuthTokens) (OAuthUser, error) {
	user, err := g.fetchUser(ctx, tokens.AccessToken)
	if err != nil {
		return OAuthUser{}, err
	}

	emails, _ := g.fetchEmails(ctx, tokens.AccessToken) // Best effort
	user.Emails = emails
	user.Data = map[string]any{"access_token": tokens.AccessToken}

	return user, nil
}

func (g *GitHubOAuth) fetchUser(ctx context.Context, token string) (OAuthUser, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return OAuthUser{}, fmt.Errorf("%w: %v", ErrOAuthUserFetch, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return OAuthUser{}, fmt.Errorf("%w: status %d", ErrOAuthUserFetch, resp.StatusCode)
	}

	var data struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return OAuthUser{}, fmt.Errorf("%w: decode: %v", ErrOAuthUserFetch, err)
	}

	return OAuthUser{
		ID:        fmt.Sprintf("%d", data.ID),
		Provider:  "github",
		Username:  data.Login,
		Name:      data.Name,
		AvatarURL: data.AvatarURL,
	}, nil
}

func (g *GitHubOAuth) fetchEmails(ctx context.Context, token string) ([]EmailAddress, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var emails []EmailAddress
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return nil, err
	}
	return emails, nil
}

func (g *GitHubOAuth) RevokeToken(ctx context.Context, tokens OAuthTokens) error {
	body, _ := json.Marshal(map[string]string{"access_token": tokens.AccessToken})
	req, _ := http.NewRequestWithContext(ctx, "DELETE",
		fmt.Sprintf("https://api.github.com/applications/%s/token", g.ClientID),
		strings.NewReader(string(body)))
	req.SetBasicAuth(g.ClientID, g.ClientSecret)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrTokenRevoke, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("%w: status %d", ErrTokenRevoke, resp.StatusCode)
	}
	return nil
}

// GitLabOAuth implements OAuth for GitLab with PKCE support.
type GitLabOAuth struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	BaseURL      string
	Scopes       string
	httpClient   *http.Client
}

func NewGitLabOAuth(clientID, clientSecret, redirectURI string) *GitLabOAuth {
	return &GitLabOAuth{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  redirectURI,
		BaseURL:      "https://gitlab.com",
		Scopes:       "openid email",
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (g *GitLabOAuth) Provider() string { return "gitlab" }

func (g *GitLabOAuth) AuthorizationURL(state, pkceChallenge string) string {
	params := url.Values{
		"client_id":     {g.ClientID},
		"redirect_uri":  {g.RedirectURI},
		"response_type": {"code"},
		"state":         {state},
		"scope":         {g.Scopes},
	}
	if pkceChallenge != "" {
		params.Set("code_challenge", pkceChallenge)
		params.Set("code_challenge_method", "S256")
	}
	return g.BaseURL + "/oauth/authorize?" + params.Encode()
}

func (g *GitLabOAuth) ExchangeCode(ctx context.Context, code, pkceVerifier string) (OAuthTokens, error) {
	data := url.Values{
		"client_id":     {g.ClientID},
		"client_secret": {g.ClientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {g.RedirectURI},
	}
	if pkceVerifier != "" {
		data.Set("code_verifier", pkceVerifier)
	}

	req, _ := http.NewRequestWithContext(ctx, "POST", g.BaseURL+"/oauth/token", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return OAuthTokens{}, fmt.Errorf("%w: %v", ErrOAuthExchange, err)
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Error        string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return OAuthTokens{}, fmt.Errorf("%w: decode: %v", ErrOAuthExchange, err)
	}
	if result.Error != "" {
		return OAuthTokens{}, fmt.Errorf("%w: %s", ErrOAuthExchange, result.Error)
	}

	tokens := OAuthTokens{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	}
	if result.ExpiresIn > 0 {
		exp := time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
		tokens.ExpiresAt = &exp
	}
	return tokens, nil
}

func (g *GitLabOAuth) RefreshToken(ctx context.Context, refreshToken string) (OAuthTokens, error) {
	data := url.Values{
		"client_id":     {g.ClientID},
		"client_secret": {g.ClientSecret},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
		"redirect_uri":  {g.RedirectURI},
	}

	req, _ := http.NewRequestWithContext(ctx, "POST", g.BaseURL+"/oauth/token", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return OAuthTokens{}, fmt.Errorf("%w: %v", ErrOAuthExchange, err)
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return OAuthTokens{}, err
	}

	tokens := OAuthTokens{AccessToken: result.AccessToken, RefreshToken: result.RefreshToken}
	if result.ExpiresIn > 0 {
		exp := time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
		tokens.ExpiresAt = &exp
	}
	return tokens, nil
}

func (g *GitLabOAuth) GetUser(ctx context.Context, tokens OAuthTokens) (OAuthUser, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", g.BaseURL+"/api/v4/user", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return OAuthUser{}, fmt.Errorf("%w: %v", ErrOAuthUserFetch, err)
	}
	defer resp.Body.Close()

	var data struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
		Email     string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return OAuthUser{}, fmt.Errorf("%w: decode: %v", ErrOAuthUserFetch, err)
	}

	user := OAuthUser{
		ID:        fmt.Sprintf("%d", data.ID),
		Provider:  "gitlab",
		Username:  data.Username,
		Name:      data.Name,
		AvatarURL: data.AvatarURL,
		Data: map[string]any{
			"access_token":  tokens.AccessToken,
			"refresh_token": tokens.RefreshToken,
		},
	}
	if data.Email != "" {
		user.Emails = []EmailAddress{{Email: data.Email, Primary: true, Verified: true}}
	}
	return user, nil
}

func (g *GitLabOAuth) RevokeToken(ctx context.Context, tokens OAuthTokens) error {
	data := url.Values{
		"client_id":     {g.ClientID},
		"client_secret": {g.ClientSecret},
		"token":         {tokens.AccessToken},
	}

	req, _ := http.NewRequestWithContext(ctx, "POST", g.BaseURL+"/oauth/revoke", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrTokenRevoke, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", ErrTokenRevoke, resp.StatusCode)
	}
	return nil
}

// PKCE helpers

// GeneratePKCE returns a (verifier, challenge) pair for PKCE OAuth flows.
func GeneratePKCE() (verifier, challenge string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h[:])
	return verifier, challenge, nil
}
