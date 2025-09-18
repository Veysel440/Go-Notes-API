package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/Veysel440/go-notes-api/internal/middleware"
	"github.com/Veysel440/go-notes-api/internal/repos"
	"github.com/go-chi/chi/v5"
)

type Notes struct{ Repo *repos.Notes }

func (h Notes) list(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.UserID(r.Context())
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	q := r.URL.Query().Get("q")
	sort := r.URL.Query().Get("sort") // newest|oldest|title|updated

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	items, total, err := h.Repo.ListFiltered(ctx, uid, page, size, q, sort)
	if err != nil {
		http.Error(w, "server", 500)
		return
	}

	var maxT time.Time
	for _, it := range items {
		if it.UpdatedAt.After(maxT) {
			maxT = it.UpdatedAt
		}
	}
	etag := fmt.Sprintf(`W/"notes-%d-%d-%d"`, maxT.Unix(), len(items), total)
	if inm := r.Header.Get("If-None-Match"); inm != "" && inm == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("ETag", etag)

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": items, "page": page, "size": size, "total": total,
	})
}

func (h Notes) get(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.UserID(r.Context())
	id64, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	n, err := h.Repo.Get(ctx, uid, id64)
	if err != nil {
		http.Error(w, "not found", 404)
		return
	}

	sum := sha256.Sum256([]byte(n.Title + "|" + n.Body))
	etag := fmt.Sprintf(`W/"n-%d-%d-%s"`, n.ID, n.UpdatedAt.Unix(), hex.EncodeToString(sum[:8]))
	if inm := r.Header.Get("If-None-Match"); inm != "" && inm == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("ETag", etag)

	_ = json.NewEncoder(w).Encode(n)
}

func (h Notes) create(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.UserID(r.Context())

	var in struct{ Title, Body string }
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad json", 400)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	id, err := h.Repo.Create(ctx, uid, in.Title, in.Body)
	if err != nil {
		http.Error(w, "server", 500)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"id": id})
}

func (h Notes) update(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.UserID(r.Context())
	id64, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	key := r.Header.Get("Idempotency-Key")
	var raw []byte
	if key != "" {
		raw, _ = io.ReadAll(io.LimitReader(r.Body, 1<<20))
		r.Body = io.NopCloser(bytes.NewReader(raw))
	}
	var in struct{ Title, Body string }
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad json", 400)
		return
	}

	if key != "" {
		sum := sha256.Sum256(raw)
		hash := hex.EncodeToString(sum[:])
		idem := repos.Idempotency{DB: h.Repo.DB}
		if doneID, err := idem.Claim(r.Context(), key, uid, "PUT", "/notes/"+strconv.FormatInt(id64, 10), hash); err != nil {
			if err == repos.ErrMismatch {
				http.Error(w, "idempotency_key_mismatch", 409)
				return
			}
			if err == repos.ErrInProgress {
				w.Header().Set("Retry-After", "2")
				http.Error(w, "idempotency_in_progress", 409)
				return
			}
			http.Error(w, "server", 500)
			return
		} else if doneID != nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		defer idem.Complete(r.Context(), key, uid, id64)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := h.Repo.Update(ctx, uid, id64, in.Title, in.Body); err != nil {
		http.Error(w, "not found", 404)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Notes) delete(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.UserID(r.Context())
	id64, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	if key := r.Header.Get("Idempotency-Key"); key != "" {
		idem := repos.Idempotency{DB: h.Repo.DB}
		if doneID, err := idem.Claim(r.Context(), key, uid, "DELETE", "/notes/"+strconv.FormatInt(id64, 10), "delete"); err != nil {
			if err == repos.ErrMismatch {
				http.Error(w, "idempotency_key_mismatch", 409)
				return
			}
			if err == repos.ErrInProgress {
				w.Header().Set("Retry-After", "2")
				http.Error(w, "idempotency_in_progress", 409)
				return
			}
			http.Error(w, "server", 500)
			return
		} else if doneID != nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		defer idem.Complete(r.Context(), key, uid, id64)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := h.Repo.Delete(ctx, uid, id64); err != nil {
		http.Error(w, "not found", 404)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Notes) Routes(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Route("/{id}", func(rr chi.Router) {
		rr.Get("/", h.get)
		rr.Put("/", h.update)
		rr.Delete("/", h.delete)
	})
}
