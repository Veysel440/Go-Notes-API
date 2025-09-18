package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/Veysel440/go-notes-api/internal/errors"
)

func RecoverJSON(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic", slog.Any("err", rec), slog.String("stack", string(debug.Stack())))
					errors.Write(w, log, r, errors.AppError{Status: 500, Code: "panic", Msg: "internal error"})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
