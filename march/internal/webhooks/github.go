package webhooks

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"july/internal/db"
	"july/internal/services"

	"github.com/jackc/pgx/v5/pgxpool"
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
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	FullName       string `json:"full_name"`
	HTMLURL        string `json:"html_url"`
	Description    string `json:"description"`
	DefaultBranch  string `json:"default_branch"`
	Private        bool   `json:"private"`
	Fork           bool   `json:"fork"`
	ForksCount     int    `json:"forks_count"`
	Watchers       int    `json:"watchers_count"`
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
	pool        *pgxpool.Pool
	gameService *services.GameService
	l1Scanner   *services.L1Scanner
}

func NewHandler(queries *db.Queries, pool *pgxpool.Pool, gameService *services.GameService, githubToken string) *Handler {
	return &Handler{
		queries:     queries,
		pool:        pool,
		gameService: gameService,
		l1Scanner:   services.NewL1Scanner(pool, githubToken),
	}
}

func (h *Handler) HandleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.Ctx(ctx)

	// Handle ping event
	if r.Header.Get("X-GitHub-Event") == "ping" {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("pong"))
		return
	}

	var data []byte
	rawContentType := r.Header.Get("Content-Type")
	contentType, _, _ := mime.ParseMediaType(rawContentType)
	hookLog := logger.With().Str("contentType", contentType).Logger()

	hookLog.Info().Msg("processing webhook body")

	switch {
	case strings.Contains(contentType, "application/json"):
		var err error
		data, err = io.ReadAll(r.Body)
		if err != nil {
			hookLog.Warn().Err(err).Msg("failed to read request body")
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}
		// smee.io wraps the payload in {"payload": "<json string>"}
		var wrapper struct {
			Payload string `json:"payload"`
		}
		if err := json.Unmarshal(data, &wrapper); err == nil && wrapper.Payload != "" {
			hookLog.Debug().Msg("unwrapping smee payload")
			data = []byte(wrapper.Payload)
		}
	case strings.Contains(contentType, "form"):
		if err := r.ParseForm(); err != nil {
			hookLog.Warn().Err(err).Msg("failed to parse form")
			http.Error(w, "failed to parse form", http.StatusBadRequest)
			return
		}
		payload := r.FormValue("payload")
		hookLog.Info().Msgf("body: %s", payload)
		if payload == "" {
			hookLog.Warn().Msg("form post missing 'payload'")
			http.Error(w, "missing payload field", http.StatusBadRequest)
			return
		}
		data = []byte(payload)
	default:
		hookLog.Warn().Msgf("unsupported content type: %s", contentType)
		http.Error(w, fmt.Sprintf("unsupported content type: %s", contentType), http.StatusUnsupportedMediaType)
		return
	}

	eventName := r.Header.Get("X-GitHub-Event")
	if eventName != "" && eventName != "push" {
		w.WriteHeader(http.StatusOK)
		hookLog.Info().Str("event", eventName).Msg("ignoring non-push webhook")
		json.NewEncoder(w).Encode(map[string]string{"status": "skipped", "reason": "not a push event"})
		return
	}

	var event GitHubPushEvent
	if err := json.Unmarshal(data, &event); err != nil {
		hookLog.Error().Err(err).Msg("failed to parse webhook payload")
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	// Skip pushes not targeting the repo default branch (see repository.default_branch in payload).
	if !isPushToDefaultBranch(event.Ref, event.Repository) {
		w.WriteHeader(http.StatusOK)
		hookLog.Info().
			Str("ref", event.Ref).
			Str("default_branch", event.Repository.DefaultBranch).
			Msg("skipping push: not the repository default branch")
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

	h.scheduleL1Scan(project)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) scheduleL1Scan(project db.Project) {
	if h.l1Scanner == nil || h.pool == nil {
		log.Warn().Msg("L1 scan skipped: handler has no pool or scanner")
		return
	}
	if !h.l1Scanner.IsConfigured() {
		log.Warn().
			Str("project", project.Slug).
			Msg("L1 scan skipped: set GITHUB_TOKEN for server-side analysis (public repos)")
		return
	}
	if project.IsPrivate {
		log.Info().Str("project", project.Slug).Msg("L1 scan skipped: private repository")
		return
	}
	proj := project
	log.Info().Str("project", proj.Slug).Str("id", proj.ID.String()).Msg("L1 scan starting in background")
	go func() {
		start := time.Now()
		defer func() {
			if r := recover(); r != nil {
				log.Error().Interface("panic", r).Str("project", proj.Slug).Msg("L1 scan panic")
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := h.l1Scanner.RunL1Scan(ctx, proj, db.SystemUserID); err != nil {
			log.Error().Err(err).Str("slug", proj.Slug).Dur("duration", time.Since(start)).Msg("L1 scan failed")
			return
		}
		log.Info().Str("slug", proj.Slug).Dur("duration", time.Since(start)).Msg("L1 scan completed")
	}()
}

func githubSlug(fullName string) string {
	return "gh-" + strings.ReplaceAll(fullName, "/", "-")
}

func (h *Handler) upsertProject(ctx context.Context, repo GitHubRepo) (db.Project, error) {
	return h.queries.UpsertProjectByRepoID(ctx, db.UpsertProjectByRepoIDParams{
		ID:          db.NewID(),
		Url:         repo.HTMLURL,
		Name:        repo.Name,
		Slug:        githubSlug(repo.FullName),
		Description: db.Text(repo.Description),
		RepoID:      db.BigInt(repo.ID),
		Service:     "github",
		Forked:      repo.Fork,
		Forks:       int32(repo.ForksCount),
		Watchers:    int32(repo.Watchers),
		IsPrivate:   repo.Private,
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
			logger.Info().Str("hash", c.ID).Str("message", c.Message).Msg("skipping invalid commit")
			result.Skipped++
			continue
		}

		_, err := h.queries.GetCommitByHash(ctx, db.Text(c.ID))
		if err == nil {
			logger.Info().Str("hash", c.ID).Msg("skipping duplicate commit")
			result.Skipped++
			continue
		}
		logger.Info().Str("hash", c.ID).Msg("commit not found, will create")

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

// isPushToDefaultBranch reports whether ref is the push for the repo's default branch.
// GitHub sets repository.default_branch (e.g. main, master, develop).
func isPushToDefaultBranch(ref string, repo GitHubRepo) bool {
	db := strings.TrimSpace(repo.DefaultBranch)
	if db == "" {
		return ref == "refs/heads/main" || ref == "refs/heads/master"
	}
	return ref == "refs/heads/"+db
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
