package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	apperr "github.com/Veysel440/go-notes-api/internal/errors"
)

type AdminJTI struct {
	Store interface {
		Revoke(ctx context.Context, jti string, ttl time.Duration) error
	}
}

func (h AdminJTI) Revoke(w http.ResponseWriter, r *http.Request) {
	var in struct {
		JTI    string `json:"jti"`
		TTLsec int    `json:"ttl_sec"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.JTI == "" {
		apperr.Write(w, r, apperr.BadRequest)
		return
	}
	ttl := time.Duration(in.TTLsec) * time.Second
	if ttl <= 0 {
		ttl = time.Hour
	}
	if err := h.Store.Revoke(r.Context(), in.JTI, ttl); err != nil {
		apperr.Write(w, r, apperr.E(500, "revoke_failed", "revoke failed", err, nil))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
