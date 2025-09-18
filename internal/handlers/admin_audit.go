package handlers

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/Veysel440/go-notes-api/internal/config"
	"github.com/Veysel440/go-notes-api/internal/repos"
)

type AdminAudit struct {
	Cfg   config.Config
	Audit *repos.Audit
}

func (h AdminAudit) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit == 0 {
		limit = 100
	}

	to := time.Now()
	if s := q.Get("to"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			to = t
		}
	}
	from := to.Add(-24 * time.Hour)
	if s := q.Get("from"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			from = t
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.Cfg.DBTimeout)
	defer cancel()

	rows, err := h.Audit.List(ctx, from, to, limit)
	if err != nil {
		http.Error(w, "server", 500)
		return
	}

	if q.Get("format") == "csv" {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="audit.csv"`)
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"id", "user_id", "method", "path", "status", "ip", "rid", "created_at"})
		for _, a := range rows {
			uid := ""
			if a.UserID.Valid {
				uid = strconv.FormatInt(a.UserID.Int64, 10)
			}
			rec := []string{
				strconv.FormatInt(a.ID, 10), uid, a.Method, a.Path,
				strconv.Itoa(a.Status), a.IP, a.RID, a.CreatedAt.Format(time.RFC3339),
			}
			_ = cw.Write(rec)
		}
		cw.Flush()
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"from": from.Format(time.RFC3339), "to": to.Format(time.RFC3339),
		"limit": limit, "data": rows,
	})
}
