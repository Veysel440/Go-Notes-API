package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type Health struct{ DB *sql.DB }

func (h Health) Live(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }
func (h Health) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.DB.PingContext(r.Context()); err != nil {
		http.Error(w, "db down", 503)
		return
	}
	w.WriteHeader(204)
}
func (h Health) Info(w http.ResponseWriter, r *http.Request) {
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}
