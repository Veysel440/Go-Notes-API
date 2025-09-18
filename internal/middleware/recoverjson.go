package middleware

import (
	"net/http"
	"runtime/debug"

	apperr "github.com/Veysel440/go-notes-api/internal/errors"
	"log/slog"
)

func RecoverJSON(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic", slog.Any("err", rec), slog.String("stack", string(debug.Stack())))
					apperr.Write(w, r, apperr.E(500, "panic", "internal error", nil, nil))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
