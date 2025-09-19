package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	apperr "github.com/Veysel440/go-notes-api/internal/errors"
	"github.com/Veysel440/go-notes-api/internal/middleware"
	"github.com/Veysel440/go-notes-api/internal/repos"
	"github.com/go-chi/chi/v5"
)

type Notes struct{ Repo *repos.Notes }

func (h Notes) Routes(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Route("/{id}", func(rr chi.Router) {
		rr.Get("/", h.get)
		rr.Put("/", h.update)
		rr.Delete("/", h.delete)
	})
}

func collETag(page, size int, q, sort string, items []repos.Note) string {
	var maxID int64
	for i := range items {
		if items[i].ID > maxID {
			maxID = items[i].ID
		}
	}
	sum := crc32.ChecksumIEEE([]byte(strings.ToLower(q) + "|" + sort))
	return fmt.Sprintf(`W/"notes-%d-%d-%d-%d-%08x"`, maxID, len(items), page, size, sum)
}

func noteETag(n repos.Note) string {
	ts := n.UpdatedAt
	if ts.IsZero() {
		ts = n.CreatedAt
	}
	h := sha256.Sum256([]byte(n.Title + "|" + n.Body))
	return fmt.Sprintf(`W/"n-%d-%d-%s"`, n.ID, ts.Unix(), hex.EncodeToString(h[:4]))
}

func (h Notes) list(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.UserID(r.Context())
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size < 1 || size > 100 {
		size = 20
	}
	q := r.URL.Query().Get("q")
	sort := r.URL.Query().Get("sort")

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	items, total, err := h.Repo.ListFiltered(ctx, uid, page, size, q, sort)
	if err != nil {
		apperr.Write(w, r, apperr.E(500, "db_error", "db error", err, nil))
		return
	}

	etag := collETag(page, size, q, sort, items)
	if inm := r.Header.Get("If-None-Match"); inm != "" && inm == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "private, max-age=0, must-revalidate")

	_ = json.NewEncoder(w).Encode(map[string]any{
		"items": items, "total": total, "page": page, "size": size,
	})
}

func (h Notes) get(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.UserID(r.Context())
	id64, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		apperr.Write(w, r, apperr.BadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	n, err := h.Repo.Get(ctx, uid, id64)
	if err != nil {
		apperr.Write(w, r, apperr.NotFound)
		return
	}

	etag := noteETag(n)
	if inm := r.Header.Get("If-None-Match"); inm != "" && inm == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "private, max-age=0, must-revalidate")
	_ = json.NewEncoder(w).Encode(n)
}

func (h Notes) create(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.UserID(r.Context())

	var in struct{ Title, Body string }
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || strings.TrimSpace(in.Title) == "" {
		apperr.Write(w, r, apperr.BadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	id, err := h.Repo.Create(ctx, uid, strings.TrimSpace(in.Title), in.Body)
	if err != nil {
		apperr.Write(w, r, apperr.E(500, "db_error", "db error", err, nil))
		return
	}

	n, err := h.Repo.Get(ctx, uid, id)
	if err != nil {
		apperr.Write(w, r, apperr.E(500, "db_error", "db error", err, nil))
		return
	}
	w.Header().Set("ETag", noteETag(n))
	_ = json.NewEncoder(w).Encode(n)
}

func (h Notes) update(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.UserID(r.Context())
	id64, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		apperr.Write(w, r, apperr.BadRequest)
		return
	}

	key := r.Header.Get("Idempotency-Key")

	var raw []byte
	if key != "" {
		raw, _ = io.ReadAll(io.LimitReader(r.Body, 1<<20))
		r.Body = io.NopCloser(bytes.NewReader(raw))
	}
	var in struct{ Title, Body string }
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		apperr.Write(w, r, apperr.BadRequest)
		return
	}

	if key != "" {
		sum := sha256.Sum256(raw)
		hash := hex.EncodeToString(sum[:])
		idem := repos.Idem{DB: h.Repo.DB}
		if res, err := idem.Claim(r.Context(), key, uid, "PUT", "/notes/"+strconv.FormatInt(id64, 10), hash); err != nil {
			switch err {
			case repos.ErrMismatch:
				apperr.Write(w, r, apperr.Conflict)
				return
			case repos.ErrInProgress:
				w.Header().Set("Retry-After", "2")
				apperr.Write(w, r, apperr.Conflict)
				return
			default:
				apperr.Write(w, r, apperr.E(500, "idem_error", "idempotency error", err, nil))
				return
			}
		} else if res != nil {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(*res))
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	n, err := h.Repo.Update(ctx, uid, id64, strings.TrimSpace(in.Title), in.Body)
	if err != nil {
		apperr.Write(w, r, apperr.NotFound)
		return
	}

	w.Header().Set("ETag", noteETag(n))
	resp, _ := json.Marshal(n)
	if key != "" {
		idem := repos.Idem{DB: h.Repo.DB}
		_ = idem.Complete(r.Context(), key, uid, string(resp))
	}
	_, _ = w.Write(resp)
}

func (h Notes) delete(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.UserID(r.Context())
	id64, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		apperr.Write(w, r, apperr.BadRequest)
		return
	}

	key := r.Header.Get("Idempotency-Key")
	if key != "" {
		idem := repos.Idem{DB: h.Repo.DB}
		if res, err := idem.Claim(r.Context(), key, uid, "DELETE", "/notes/"+strconv.FormatInt(id64, 10), "delete"); err != nil {
			switch err {
			case repos.ErrMismatch:
				apperr.Write(w, r, apperr.Conflict)
				return
			case repos.ErrInProgress:
				w.Header().Set("Retry-After", "2")
				apperr.Write(w, r, apperr.Conflict)
				return
			default:
				apperr.Write(w, r, apperr.E(500, "idem_error", "idempotency error", err, nil))
				return
			}
		} else if res != nil {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(*res))
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	n, err := h.Repo.Delete(ctx, uid, id64)
	if err != nil {
		apperr.Write(w, r, apperr.NotFound)
		return
	}

	w.Header().Set("ETag", noteETag(n))
	resp, _ := json.Marshal(n)
	if key != "" {
		idem := repos.Idem{DB: h.Repo.DB}
		_ = idem.Complete(r.Context(), key, uid, string(resp))
	}
	_, _ = w.Write(resp)
}
