package repos

import (
	"context"
	"database/sql"
	"time"
)

type Audit struct{ DB *sql.DB }

type AuditRow struct {
	ID        int64
	UserID    sql.NullInt64
	Method    string
	Path      string
	Status    int
	IP        string
	RID       string
	CreatedAt time.Time
}

func (r Audit) List(ctx context.Context, from, to time.Time, limit int) ([]AuditRow, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, user_id, method, path, status, ip, rid, created_at
		FROM audit_logs
		WHERE created_at BETWEEN ? AND ?
		ORDER BY id DESC
		LIMIT ?`, from, to, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AuditRow
	for rows.Next() {
		var a AuditRow
		if err := rows.Scan(&a.ID, &a.UserID, &a.Method, &a.Path, &a.Status, &a.IP, &a.RID, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
