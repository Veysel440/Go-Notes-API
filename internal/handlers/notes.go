package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Veysel440/go-notes-api/internal/middleware"
	"github.com/Veysel440/go-notes-api/internal/repos"
	"github.com/go-chi/chi/v5"
)

type Notes struct{ Repo *repos.Notes }

func (h Notes) Routes(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Get("/{id}", h.get)
	r.Put("/{id}", h.update)
	r.Delete("/{id}", h.del)
}

func (h Notes) list(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.UserID(r.Context())
	items, _ := h.Repo.List(r.Context(), uid)
	_ = json.NewEncoder(w).Encode(items)
}
func (h Notes) create(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.UserID(r.Context())
	var in struct{ Title, Body string }
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	id, _ := h.Repo.Create(r.Context(), uid, in.Title, in.Body)
	_ = json.NewEncoder(w).Encode(map[string]any{"id": id})
}
func (h Notes) get(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.UserID(r.Context())
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	item, err := h.Repo.Get(r.Context(), uid, id)
	if err != nil {
		http.Error(w, "not found", 404)
		return
	}
	_ = json.NewEncoder(w).Encode(item)
}
func (h Notes) update(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.UserID(r.Context())
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var in struct{ Title, Body string }
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	if err := h.Repo.Update(r.Context(), uid, id, in.Title, in.Body); err != nil {
		http.Error(w, "not found", 404)
		return
	}
	w.WriteHeader(204)
}
func (h Notes) del(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.UserID(r.Context())
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.Repo.Delete(r.Context(), uid, id); err != nil {
		http.Error(w, "not found", 404)
		return
	}
	w.WriteHeader(204)
}
