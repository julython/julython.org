package api

import (
	"fmt"
	"net/http"
	"os"
	"runtime/debug"

	"github.com/rs/zerolog/log"
)

// RecoveryMiddleware catches panics, logs them with the request-scoped logger,
// and prints the stack trace to stderr in dev-friendly format.
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Ctx(r.Context()).Error().
					Interface("error", err).
					Msg("panic recovered")
				fmt.Fprintf(os.Stderr, "\n%s\n", debug.Stack())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
