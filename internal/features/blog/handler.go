package blog

import (
	"fmt"
	"net/http"

	"july/internal/components/layout"
	"july/internal/handlers"

	"github.com/rs/zerolog/log"
)

// Handler handles blog-related HTTP requests.
type Handler struct{}

// NewHandler creates a new blog handler.
func NewHandler() *Handler {
	return &Handler{}
}

// Register mounts blog routes on the given mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /blog", h.List)
	mux.HandleFunc("GET /blog/{slug}", h.Detail)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.Ctx(ctx)

	posts, err := All()
	if err != nil {
		logger.Info().Err(err).Msg("Error finding blogs")
		http.Error(w, "blogs not found", http.StatusNotFound)
		return
	}

	layout := layout.LayoutData{
		Title:       "Blog",
		CurrentPath: "/blog",
		User:        userInfoFromContext(r),
	}

	BlogListPage(layout, posts).Render(ctx, w)
}

func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.Ctx(ctx)

	slug := r.PathValue("slug")
	post, err := BySlug(slug)
	if err != nil {
		logger.Info().Err(err).Msg("Error finding blog")
		msg := fmt.Sprintf("blog %s not found", slug)
		http.Error(w, msg, http.StatusNotFound)
		return
	}

	layout := layout.LayoutData{
		Title:       post.Title,
		CurrentPath: fmt.Sprintf("/blog/%s", slug),
		User:        userInfoFromContext(r),
	}

	BlogPostPage(layout, post).Render(ctx, w)
}

func userInfoFromContext(r *http.Request) *layout.UserInfo {
	u := handlers.UserFromContext(r.Context())
	if u == nil {
		return nil
	}
	return &layout.UserInfo{
		Username:  u.Username,
		AvatarURL: u.AvatarURL,
	}
}
