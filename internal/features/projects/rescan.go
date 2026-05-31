package projects

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"july/internal/auth"
	"july/internal/db"
)

var (
	errL1PrivateRepo   = errors.New("l1: private repository")
	errL1NoGitHubToken = errors.New("l1: GITHUB_TOKEN not configured")
)

// performL1Scan runs server-side L1 (push webhook and manual rescan use this path).
func (h *projectHandler) performL1Scan(ctx context.Context, project db.Project, updatedBy uuid.UUID) error {
	if project.IsPrivate {
		return errL1PrivateRepo
	}
	if !h.l1Scanner.IsConfigured() {
		return errL1NoGitHubToken
	}

	scanCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if err := h.l1Scanner.RunL1Scan(scanCtx, project, updatedBy); err != nil {
		return err
	}
	return nil
}

// POST /projects/{slug}/analysis/l1
// HTMX-triggered L1 rescan from the project page; redirects back on success.
func (h *projectHandler) PostProjectRescanL1(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slug := r.PathValue("slug")

	sessionUser := auth.UserFromContext(ctx)
	if sessionUser == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	user, err := h.userService.FindByID(ctx, sessionUser.ID)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	project, err := h.queries.GetProjectBySlug(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("slug", slug).Msg("get project for rescan")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if !canEditProject(&user, project) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	err = h.performL1Scan(ctx, project, user.ID)
	if errors.Is(err, errL1PrivateRepo) {
		http.Error(w, "server L1 is not available for private repositories", http.StatusBadRequest)
		return
	}
	if errors.Is(err, errL1NoGitHubToken) {
		http.Error(w, "GITHUB_TOKEN is not configured", http.StatusServiceUnavailable)
		return
	}
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("slug", slug).Msg("L1 scan")
		http.Error(w, "L1 scan failed", http.StatusBadGateway)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", fmt.Sprintf("/projects/%s", slug))
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/projects/%s", slug), http.StatusSeeOther)
}
