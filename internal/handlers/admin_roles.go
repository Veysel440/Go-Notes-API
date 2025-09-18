package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Veysel440/go-notes-api/internal/config"
	"github.com/Veysel440/go-notes-api/internal/repos"
	"github.com/go-chi/chi/v5"
)

type AdminRoles struct {
	Cfg   config.Config
	Roles *repos.Roles
}

func (h AdminRoles) Post(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	var in struct {
		Action string `json:"action"`
		Role   string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad json", 400)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.Cfg.DBTimeout)
	defer cancel()

	switch in.Action {
	case "add":
		if err := h.Roles.Assign(ctx, id, in.Role); err != nil {
			http.Error(w, "server", 500)
			return
		}
	case "remove":
		if err := h.Roles.Unassign(ctx, id, in.Role); err != nil {
			http.Error(w, "server", 500)
			return
		}
	default:
		http.Error(w, "invalid action", 422)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
