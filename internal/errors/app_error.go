package errors

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type AppError struct {
	Status    int
	Code, Msg string
}

func (e AppError) Error() string { return e.Msg }

var (
	ErrUnauthorized = AppError{Status: 401, Code: "unauthorized", Msg: "unauthorized"}
	ErrForbidden    = AppError{Status: 403, Code: "forbidden", Msg: "forbidden"}
	ErrNotFound     = AppError{Status: 404, Code: "not_found", Msg: "not found"}
)

func Write(w http.ResponseWriter, log *slog.Logger, r *http.Request, err error) {
	app, ok := err.(AppError)
	if !ok {
		app = AppError{Status: 500, Code: "internal", Msg: "internal error"}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(app.Status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": app.Code, "message": app.Msg, "rid": r.Header.Get("X-Request-ID"),
	})
	if app.Status >= 500 {
		log.Error("error", slog.String("code", app.Code), slog.String("msg", app.Msg))
	}
}
