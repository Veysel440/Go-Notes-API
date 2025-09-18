package errors

import (
	"encoding/json"
	stderrs "errors"
	"log/slog"
	"net/http"

	"github.com/Veysel440/go-notes-api/internal/logging"
)

type AppError struct {
	Status  int               `json:"-"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
	Err     error             `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func E(status int, code, msg string, cause error, fields map[string]string) *AppError {
	return &AppError{Status: status, Code: code, Message: msg, Fields: fields, Err: cause}
}

var (
	Unauthorized = &AppError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "unauthorized"}
	Forbidden    = &AppError{Status: http.StatusForbidden, Code: "forbidden", Message: "forbidden"}
	NotFound     = &AppError{Status: http.StatusNotFound, Code: "not_found", Message: "not found"}
	Conflict     = &AppError{Status: http.StatusConflict, Code: "conflict", Message: "conflict"}
	BadRequest   = &AppError{Status: http.StatusBadRequest, Code: "bad_request", Message: "bad request"}
	TooMany      = &AppError{Status: http.StatusTooManyRequests, Code: "too_many_requests", Message: "too many requests"}
	Validation   = func(fields map[string]string) *AppError {
		return &AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "validation error", Fields: fields}
	}
)

func Write(w http.ResponseWriter, r *http.Request, err error) {
	var app *AppError
	if !stderrs.As(err, &app) {
		app = &AppError{Status: http.StatusInternalServerError, Code: "internal_error", Message: "internal error", Err: err}
	}
	rid := r.Header.Get("X-Request-ID")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(app.Status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code": app.Code, "message": app.Message, "rid": rid, "fields": app.Fields,
	})

	log := logging.New()
	lvl := slog.LevelError
	if app.Status < 500 {
		lvl = slog.LevelWarn
	}
	log.LogAttrs(r.Context(), lvl, "api_error",
		slog.Int("status", app.Status),
		slog.String("code", app.Code),
		slog.String("rid", rid),
		slog.String("path", r.URL.Path),
		slog.String("cause", func() string {
			if app.Err != nil {
				return app.Err.Error()
			}
			return ""
		}()),
	)
}
