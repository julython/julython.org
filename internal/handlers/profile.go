package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"july/internal/components"
	"july/internal/services"
)

type ProfileHandler struct {
	users      *services.UserService
	session    *scs.SessionManager
	webhookURL string // e.g. "https://julython.org/webhooks/github"
}

func NewProfileHandler(
	users *services.UserService,
	session *scs.SessionManager,
	webhookURL string,
) *ProfileHandler {
	return &ProfileHandler{
		users:      users,
		session:    session,
		webhookURL: webhookURL,
	}
}

// -----------------------------------------------------------------------
// Overview
// -----------------------------------------------------------------------

func (h *ProfileHandler) Overview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess := UserFromContext(ctx)
	if sess == nil {
		http.Redirect(w, r, "/auth/login/github", http.StatusFound)
		return
	}

	emails, _ := h.users.GetVerifiedEmails(ctx, sess.ID)

	layout := h.layout(r, "Profile")
	data := components.OverviewData{
		Username:  sess.Username,
		Name:      sess.Name,
		AvatarURL: sess.AvatarURL,
		Emails:    emails,
	}
	components.OverviewPage(layout, data).Render(ctx, w)
}

// -----------------------------------------------------------------------
// Webhooks — page shell
// -----------------------------------------------------------------------

func (h *ProfileHandler) Webhooks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if UserFromContext(ctx) == nil {
		http.Redirect(w, r, "/auth/login/github", http.StatusFound)
		return
	}
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}

	components.WebhooksPage(h.layout(r, "Webhooks"), page).Render(ctx, w)
}

// -----------------------------------------------------------------------
// Webhooks — async repo list (HTMX target)
// -----------------------------------------------------------------------

const webhookReposPerPage = 10

func (h *ProfileHandler) WebhookRepos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess := UserFromContext(ctx)
	if sess == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	token, err := h.users.GetOAuthToken(ctx, sess.ID, services.IdentifierGitHub)
	if err != nil {
		http.Error(w, "no GitHub token", http.StatusUnauthorized)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}

	gh := services.NewGitHubService(token)
	ghRepos, err := gh.ListRepos(ctx, true, webhookReposPerPage+1, page)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("list github repos")
		http.Error(w, "failed to list repos", http.StatusInternalServerError)
		return
	}

	hasMore := len(ghRepos) > webhookReposPerPage
	if hasMore {
		ghRepos = ghRepos[:webhookReposPerPage]
	}

	repos := make([]services.RepoWithWebhook, len(ghRepos))
	for i, gr := range ghRepos {
		rw := services.RepoWithWebhook{GitHubRepo: gr}
		for _, wh := range gr.Webhooks {
			if strings.HasPrefix(wh.Config.URL, h.webhookURL) && wh.Active {
				rw.HasWebhook = true
				rw.WebhookID = wh.ID
				break
			}
		}
		repos[i] = rw
	}

	components.WebhookRepoList(repos, h.webhookURL, page, hasMore).Render(ctx, w)
}

// -----------------------------------------------------------------------
// Webhooks — add
// -----------------------------------------------------------------------

func (h *ProfileHandler) AddWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess := UserFromContext(ctx)
	if sess == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	repoID, err := strconv.Atoi(r.PathValue("repoID"))
	if err != nil {
		http.Error(w, "bad repo id", http.StatusBadRequest)
		return
	}

	owner := r.FormValue("owner")
	repo := r.FormValue("repo")
	if owner == "" || repo == "" {
		http.Error(w, "missing owner/repo", http.StatusBadRequest)
		return
	}

	token, err := h.users.GetOAuthToken(ctx, sess.ID, services.IdentifierGitHub)
	if err != nil {
		http.Error(w, "no GitHub token", http.StatusUnauthorized)
		return
	}

	gh := services.NewGitHubService(token)

	wh, err := gh.CreateWebhook(ctx, owner, repo, h.webhookURL, []string{"push"})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("repo", fmt.Sprintf("%s/%s", owner, repo)).Msg("create webhook")
		http.Error(w, "failed to add webhook", http.StatusInternalServerError)
		return
	}

	// Re-fetch repo details so we can return an updated row.
	repos, _ := gh.ListRepos(ctx, false, 100, 1)
	for _, gr := range repos {
		if gr.ID == repoID {
			rw := services.RepoWithWebhook{
				GitHubRepo: gr,
				HasWebhook: true,
				WebhookID:  wh.ID,
			}
			components.WebhookRepoRow(rw, h.webhookURL).Render(ctx, w)
			return
		}
	}

	// Fallback: just signal success with a minimal row.
	http.Error(w, "webhook created but repo not found in list", http.StatusOK)
}

// -----------------------------------------------------------------------
// Webhooks — delete
// -----------------------------------------------------------------------

func (h *ProfileHandler) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess := UserFromContext(ctx)
	if sess == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	repoID, err := strconv.Atoi(r.PathValue("repoID"))
	if err != nil {
		http.Error(w, "bad repo id", http.StatusBadRequest)
		return
	}
	hookID, err := strconv.Atoi(r.PathValue("hookID"))
	if err != nil {
		http.Error(w, "bad hook id", http.StatusBadRequest)
		return
	}

	token, err := h.users.GetOAuthToken(ctx, sess.ID, services.IdentifierGitHub)
	if err != nil {
		http.Error(w, "no GitHub token", http.StatusUnauthorized)
		return
	}

	gh := services.NewGitHubService(token)

	// Find the repo so we have owner/name for the API call.
	repos, err := gh.ListRepos(ctx, false, 100, 1)
	if err != nil {
		http.Error(w, "failed to list repos", http.StatusInternalServerError)
		return
	}

	for _, gr := range repos {
		if gr.ID != repoID {
			continue
		}
		if err := gh.DeleteWebhook(ctx, gr.Owner.Login, gr.Name, hookID); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("delete webhook")
			http.Error(w, "failed to delete webhook", http.StatusInternalServerError)
			return
		}
		rw := services.RepoWithWebhook{GitHubRepo: gr}
		components.WebhookRepoRow(rw, h.webhookURL).Render(ctx, w)
		return
	}

	http.Error(w, "repo not found", http.StatusNotFound)
}

// -----------------------------------------------------------------------
// Settings — GET
// -----------------------------------------------------------------------

func (h *ProfileHandler) Settings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess := UserFromContext(ctx)
	if sess == nil {
		http.Redirect(w, r, "/auth/login/github", http.StatusFound)
		return
	}
	components.SettingsPage(h.layout(r, "Settings"), components.SettingsData{
		Name: sess.Name,
	}).Render(ctx, w)
}

// -----------------------------------------------------------------------
// Settings — POST (works plain or via HTMX)
// -----------------------------------------------------------------------

func (h *ProfileHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sess := UserFromContext(ctx)
	if sess == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Support both form-encoded and JSON bodies.
	name, errMsg := settingsNameFromRequest(r)

	data := components.SettingsData{Name: name}

	if errMsg != "" {
		data.Error = errMsg
	} else {
		userID, err := uuid.Parse(sess.ID.String())
		if err == nil {
			_, err = h.users.UpdateProfile(ctx, userID, name)
		}
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("update profile")
			data.Error = "Failed to save. Please try again."
		} else {
			sess.Name = name
			sess.Name = name
			// persist back to the session store
			h.session.Put(ctx, "user_name", name)
			data.Success = true
		}
	}

	isHTMX := r.Header.Get("HX-Request") == "true"
	if isHTMX {
		// Return just the form fragment so HTMX can swap it.
		w.Header().Set("Content-Type", "text/html")
		components.SettingsFormFragment(data).Render(ctx, w)
		return
	}

	components.SettingsPage(h.layout(r, "Settings"), data).Render(ctx, w)
}

// -----------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------

func (h *ProfileHandler) layout(r *http.Request, title string) components.LayoutData {
	return components.LayoutData{
		Title:       title,
		CurrentPath: r.URL.Path,
		User:        getUserFromContext(r),
	}
}

func settingsNameFromRequest(r *http.Request) (name, errMsg string) {
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "application/json") {
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return "", "Invalid request body."
		}
		name = strings.TrimSpace(body.Name)
	} else {
		if err := r.ParseForm(); err != nil {
			return "", "Invalid form data."
		}
		name = strings.TrimSpace(r.FormValue("name"))
	}

	if name == "" {
		return name, "Name cannot be empty."
	}
	if len(name) > 120 {
		return name, "Name must be 120 characters or fewer."
	}
	return name, ""
}
