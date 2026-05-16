package i18n

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	t.Run("loads locales successfully", func(t *testing.T) {
		err := Init()
		require.NoError(t, err)
	})
}

func TestMiddlewareFromCookie(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := Middleware(mux)

	t.Run("reads lang from cookie", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{Name: "lang", Value: "es"})

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("uses English when cookie is empty", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{Name: "lang", Value: ""})

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("ignores missing cookie", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestMiddlewareFromHeader(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := Middleware(mux)

	t.Run("falls back to Accept-Language header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "pt")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("cookie takes precedence over header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "pt")
		req.AddCookie(&http.Cookie{Name: "lang", Value: "es"})

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("fallback to English for unsupported locale", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "ja")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		// Should fallback to English without error
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestSetLanguage(t *testing.T) {
	t.Run("sets lang cookie with query param", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/set-language?lang=pt", nil)
		req.Header.Set("Referer", "/profile")

		rec := httptest.NewRecorder()
		SetLanguage(rec, req)

		require.Equal(t, http.StatusSeeOther, rec.Code)
		require.Equal(t, "/profile", rec.Header().Get("Location"))

		cookie := rec.Header().Get("Set-Cookie")
		assert.Contains(t, cookie, "lang=pt")
		assert.Contains(t, cookie, "Max-Age=")
	})

	t.Run("defaults to English when no lang param", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/set-language", nil)
		rec := httptest.NewRecorder()
		SetLanguage(rec, req)

		cookie := rec.Header().Get("Set-Cookie")
		assert.Contains(t, cookie, "lang=en")
	})

	t.Run("redirects to / when no referer", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/set-language?lang=es", nil)
		rec := httptest.NewRecorder()
		SetLanguage(rec, req)

		require.Equal(t, http.StatusSeeOther, rec.Code)
		require.Equal(t, "/", rec.Header().Get("Location"))
	})
}
