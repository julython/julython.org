// Package web embeds static assets and exposes HTTP handlers for serving them.
// In production (DEV != "1") assets are served from the embedded FS baked into
// the binary. In development they are served from disk so that tailwind --watch
// changes are visible without a full Go rebuild.
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
)

//go:embed all:assets all:static
var files embed.FS

// AssetsHandler serves /assets/ content.
// Hashed filenames (e.g. tailwind.abc12345.css) allow infinite caching in prod.
// Register as: mux.Handle("GET /assets/", http.StripPrefix("/assets/", web.AssetsHandler()))
func AssetsHandler() http.Handler {
	return cacheHandler(diskOrEmbed("assets"), true)
}

// StaticHandler serves /static/ content with shorter caching.
// Register as: mux.Handle("GET /static/", http.StripPrefix("/static/", web.StaticHandler()))
func StaticHandler() http.Handler {
	return cacheHandler(diskOrEmbed("static"), false)
}

func diskOrEmbed(dir string) http.Handler {
	if os.Getenv("DEV") == "1" {
		// Serve from disk so file changes are visible without rebuilding.
		return http.FileServer(http.Dir("web/" + dir))
	}
	sub, _ := fs.Sub(files, dir)
	return http.FileServer(http.FS(sub))
}

func cacheHandler(h http.Handler, immutable bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case os.Getenv("DEV") == "1":
			w.Header().Set("Cache-Control", "no-cache")
		case immutable:
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		default:
			w.Header().Set("Cache-Control", "public, max-age=3600")
		}
		h.ServeHTTP(w, r)
	})
}
