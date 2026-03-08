package webhooks

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"july/internal/db"
	"july/internal/services"

	"github.com/rs/zerolog/log"
)

var (
	ErrInvalidSignature = errors.New("invalid webhook signature")
	ErrUnknownRepo      = errors.New("repository not registered")
)

// GitHub webhook payload types

type GitHubPushEvent struct {
	Ref        string         `json:"ref"`
	Before     string         `json:"before"`
	After      string         `json:"after"`
	Forced     bool           `json:"forced"`
	Repository GitHubRepo     `json:"repository"`
	Commits    []GitHubCommit `json:"commits"`
}

type GitHubRepo struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	HTMLURL     string `json:"html_url"`
	Description string `json:"description"`
	Fork        bool   `json:"fork"`
	ForksCount  int    `json:"forks_count"`
	Watchers    int    `json:"watchers_count"`
}

type GitHubCommit struct {
	ID        string       `json:"id"`
	Message   string       `json:"message"`
	Timestamp time.Time    `json:"timestamp"`
	URL       string       `json:"url"`
	Author    GitHubAuthor `json:"author"`
	Added     []string     `json:"added"`
	Modified  []string     `json:"modified"`
	Removed   []string     `json:"removed"`
}

type GitHubAuthor struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

// FileChange represents a file modification in a commit
type FileChange struct {
	File     string `json:"file"`
	Type     string `json:"type"`
	Language string `json:"language,omitempty"`
}

type Handler struct {
	queries     *db.Queries
	gameService *services.GameService
}

func NewHandler(queries *db.Queries, gameService *services.GameService) *Handler {
	return &Handler{
		queries:     queries,
		gameService: gameService,
	}
}

func (h *Handler) HandleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.Ctx(ctx)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	var event GitHubPushEvent
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Skip non-default branches
	if !isDefaultBranch(event.Ref) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "skipped", "reason": "not default branch"})
		return
	}

	// Skip force pushes (potential history rewrite)
	if event.Forced {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "skipped", "reason": "force push"})
		return
	}

	// Upsert project
	project, err := h.upsertProject(ctx, event.Repository)
	if err != nil {
		logger.Error().
			Err(err).
			Str("repo", event.Repository.FullName).
			Msg("failed to upsert project")
		http.Error(w, "failed to process repository", http.StatusInternalServerError)
		return
	}

	if !project.IsActive {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "skipped", "reason": "project inactive"})
		return
	}

	result, err := h.processCommits(ctx, project, event.Commits)
	if err != nil {
		logger.Error().
			Err(err).
			Str("project", project.Slug).
			Str("name", project.Name).
			Msg("failed to process commits")
		http.Error(w, "failed to process commits", http.StatusInternalServerError)
		return
	}

	logger.Info().
		Str("project", project.Slug).
		Int("received", result.Received).
		Int("created", result.Created).
		Int("skipped", result.Skipped).
		Msg("processed webhook")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) upsertProject(ctx context.Context, repo GitHubRepo) (db.Project, error) {
	return h.queries.UpsertProjectByRepoID(ctx, db.UpsertProjectByRepoIDParams{
		ID:          db.NewID(),
		Url:         repo.HTMLURL,
		Name:        repo.Name,
		Slug:        repo.FullName,
		Description: db.Text(repo.Description),
		RepoID:      db.BigInt(repo.ID),
		Service:     "github",
		Forked:      repo.Fork,
		Forks:       int32(repo.ForksCount),
		Watchers:    int32(repo.Watchers),
	})
}

type ProcessResult struct {
	Received int `json:"received"`
	Created  int `json:"created"`
	Skipped  int `json:"skipped"`
}

func (h *Handler) processCommits(ctx context.Context, project db.Project, commits []GitHubCommit) (*ProcessResult, error) {
	result := &ProcessResult{Received: len(commits)}
	logger := log.Ctx(ctx)

	for _, c := range commits {
		// Skip low-quality commits
		if !IsValidCommit(c) {
			result.Skipped++
			continue
		}

		// Check if commit already exists
		_, err := h.queries.GetCommitByHash(ctx, db.Text(c.ID))
		if err == nil {
			result.Skipped++
			continue
		}

		// Find user by email
		userID := db.NullUUID()
		if user, err := h.queries.FindUserByIdentifier(ctx, "email:"+c.Author.Email); err == nil {
			userID = db.UUID(user.ID)
		}

		// Parse files and detect languages
		files, languages := parseFiles(c.Added, c.Modified, c.Removed)
		filesJSON, _ := json.Marshal(files)

		// Create commit
		commit, err := h.queries.CreateCommit(ctx, db.CreateCommitParams{
			ID:        db.NewID(),
			Hash:      db.Text(c.ID),
			ProjectID: project.ID,
			UserID:    userID,
			GameID:    db.NullUUID(), // Will be set by GameService
			Author:    db.Text(c.Author.Name),
			Email:     db.Text(c.Author.Email),
			Message:   truncate(c.Message, 2000),
			Url:       c.URL,
			Timestamp: c.Timestamp,
			Languages: languages,
			Files:     filesJSON,
		})
		if err != nil {
			logger.Warn().
				Err(err).
				Str("hash", c.ID).
				Msg("failed to create commit")
			result.Skipped++
			continue
		}

		// Update game scores
		if err := h.gameService.AddCommit(ctx, commit); err != nil {
			logger.Error().Err(err).Str("hash", c.ID).Msg("failed to add commit to game")
			// Don't fail the whole request
		}

		result.Created++
	}

	return result, nil
}

// IsValidCommit checks if a commit should be counted for points
func IsValidCommit(c GitHubCommit) bool {
	msg := strings.ToLower(c.Message)

	// Skip very short messages
	if len(c.Message) < 5 {
		return false
	}

	// Skip WIP commits
	if strings.HasPrefix(msg, "wip") {
		return false
	}

	// Skip merge commits
	if strings.HasPrefix(msg, "merge ") {
		return false
	}

	// Skip automated commits
	if strings.Contains(msg, "[skip ci]") || strings.Contains(msg, "[ci skip]") {
		return false
	}

	return true
}

func parseFiles(added, modified, removed []string) ([]FileChange, []string) {
	var files []FileChange
	langSet := make(map[string]bool)

	addFiles := func(paths []string, changeType string) {
		for _, path := range paths {
			lang := DetectLanguage(path)
			files = append(files, FileChange{
				File:     path,
				Type:     changeType,
				Language: lang,
			})
			if lang != "" {
				langSet[lang] = true
			}
		}
	}

	addFiles(added, "added")
	addFiles(modified, "modified")
	addFiles(removed, "removed")

	languages := make([]string, 0, len(langSet))
	for lang := range langSet {
		languages = append(languages, lang)
	}

	return files, languages
}

// DetectLanguage returns the programming language based on file extension
func DetectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))

	languages := map[string]string{
		".go":     "Go",
		".py":     "Python",
		".js":     "JavaScript",
		".ts":     "TypeScript",
		".jsx":    "JavaScript",
		".tsx":    "TypeScript",
		".rs":     "Rust",
		".rb":     "Ruby",
		".java":   "Java",
		".kt":     "Kotlin",
		".swift":  "Swift",
		".c":      "C",
		".cpp":    "C++",
		".cc":     "C++",
		".h":      "C",
		".hpp":    "C++",
		".cs":     "C#",
		".php":    "PHP",
		".scala":  "Scala",
		".ex":     "Elixir",
		".exs":    "Elixir",
		".erl":    "Erlang",
		".hs":     "Haskell",
		".ml":     "OCaml",
		".clj":    "Clojure",
		".lua":    "Lua",
		".r":      "R",
		".jl":     "Julia",
		".pl":     "Perl",
		".sh":     "Shell",
		".bash":   "Shell",
		".zsh":    "Shell",
		".sql":    "SQL",
		".html":   "HTML",
		".css":    "CSS",
		".scss":   "SCSS",
		".sass":   "Sass",
		".less":   "Less",
		".vue":    "Vue",
		".svelte": "Svelte",
	}

	return languages[ext]
}

func verifySignature(body []byte, signature, secret string) error {
	if signature == "" {
		return ErrInvalidSignature
	}

	sig := strings.TrimPrefix(signature, "sha256=")
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return ErrInvalidSignature
	}
	return nil
}

func isDefaultBranch(ref string) bool {
	return ref == "refs/heads/main" || ref == "refs/heads/master"
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
