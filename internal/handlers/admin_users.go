package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Veysel440/go-notes-api/internal/config"
	"github.com/Veysel440/go-notes-api/internal/repos"
)

type AdminUsers struct {
	Cfg   config.Config
	Users *repos.Users
}

func (h AdminUsers) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))

	ctx, cancel := context.WithTimeout(r.Context(), h.Cfg.DBTimeout)
	defer cancel()

	items, total, err := h.Users.List(ctx, page, size, q)
	if err != nil {
		http.Error(w, "server", 500)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": items,
		"page": page, "size": size, "total": total,
	})
}
