package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/johnnynv/RepoSentry/pkg/logger"
)

// Recovery recovers from panics and logs them
func Recovery(log *logger.Entry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.WithFields(logger.Fields{
						"error":  err,
						"stack":  string(debug.Stack()),
						"path":   r.URL.Path,
						"method": r.Method,
					}).Error("Panic recovered in HTTP handler")

					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
