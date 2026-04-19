package handlers

import (
	"fmt"
	"net/http"

	"july/internal/blog"
	"july/internal/components"

	"github.com/rs/zerolog/log"
)

type BlogHandler struct{}

func NewBlogHandler() *BlogHandler {
	return &BlogHandler{}
}

func (h *BlogHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.Ctx(ctx)

	posts, err := blog.All()
	if err != nil {
		logger.Info().Err(err).Msg("Error finding blogs")
		http.Error(w, "blogs not found", http.StatusNotFound)
		return
	}

	layout := components.LayoutData{
		Title:       "Blog",
		CurrentPath: "/blog",
		User:        getUserFromContext(r),
	}

	components.BlogListPage(layout, posts).Render(ctx, w)
}

func (h *BlogHandler) Detail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.Ctx(ctx)

	slug := r.PathValue("slug")
	post, err := blog.BySlug(slug)
	if err != nil {
		logger.Info().Err(err).Msg("Error finding blog")
		msg := fmt.Sprintf("blog %s not found", slug)
		http.Error(w, msg, http.StatusNotFound)
		return
	}

	layout := components.LayoutData{
		Title:       post.Title,
		CurrentPath: fmt.Sprintf("/blog/%s", slug),
		User:        getUserFromContext(r),
	}

	components.BlogPostPage(layout, post).Render(ctx, w)
}
