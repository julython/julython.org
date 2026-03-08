package i18n

import (
	"embed"
	"net/http"

	"github.com/invopop/ctxi18n"
	"github.com/rs/zerolog/log"
)

//go:embed locales/*.yaml
var localeFS embed.FS

func Init() error {
	if err := ctxi18n.LoadWithDefault(localeFS, "en"); err != nil {
		return err
	}
	log.Info().Msg("i18n locales loaded")
	return nil
}

// Middleware reads the preferred locale from the "lang" cookie, falling back
// to the Accept-Language header, then "en". It adds the locale to the context
// so that ctxi18n/i18n.T(ctx, ...) works in handlers and templ components.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := "en"
		if cookie, err := r.Cookie("lang"); err == nil && cookie.Value != "" {
			lang = cookie.Value
		} else if accept := r.Header.Get("Accept-Language"); accept != "" {
			lang = accept
		}

		ctx, err := ctxi18n.WithLocale(r.Context(), lang)
		if err != nil {
			// Fallback to English if locale not found
			ctx, _ = ctxi18n.WithLocale(r.Context(), "en")
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// SetLanguage handles /set-language?lang=es, persists the choice in a cookie,
// then redirects back to the referring page.
func SetLanguage(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("lang")
	if lang == "" {
		lang = "en"
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "lang",
		Value:    lang,
		Path:     "/",
		MaxAge:   365 * 24 * 60 * 60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	ref := r.Header.Get("Referer")
	if ref == "" {
		ref = "/"
	}
	http.Redirect(w, r, ref, http.StatusSeeOther)
}
