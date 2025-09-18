package repos

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Veysel440/go-notes-api/internal/metrics"
)

type Note struct {
	ID        int64        `json:"id"`
	UserID    int64        `json:"-"`
	Title     string       `json:"title"`
	Body      string       `json:"body"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	DeletedAt sql.NullTime `json:"-"`
}

type Notes struct {
	DB *sql.DB
	Mx *metrics.Registry
}

func (r *Notes) observe(op string, start time.Time) {
	if r.Mx != nil {
		r.Mx.ObserveDB(op, time.Since(start))
	}
}

func sanitizeSort(s string) string {
	switch s {
	case "oldest":
		return "created_at ASC"
	case "title":
		return "title ASC, id DESC"
	case "updated":
		return "updated_at DESC"
	default:
		return "id DESC"
	}
}

func (r *Notes) ListFiltered(ctx context.Context, uid int64, page, size int, q, sort string) ([]Note, int64, error) {
	start := time.Now()
	defer r.observe("notes_list", start)

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	where := "WHERE user_id=? AND deleted_at IS NULL"
	args := []any{uid}
	if q != "" {
		where += " AND (title LIKE ? OR body LIKE ?)"
		like := "%" + q + "%"
		args = append(args, like, like)
	}

	var total int64
	if err := r.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM notes "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	order := sanitizeSort(sort)
	args = append(args, size, (page-1)*size)
	query := fmt.Sprintf(`
		SELECT id,user_id,title,body,created_at,updated_at,deleted_at
		FROM notes %s
		ORDER BY %s
		LIMIT ? OFFSET ?`, where, order)

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]Note, 0, size)
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Body, &n.CreatedAt, &n.UpdatedAt, &n.DeletedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, n)
	}
	return out, total, rows.Err()
}

func (r *Notes) Create(ctx context.Context, uid int64, title, body string) (int64, error) {
	start := time.Now()
	defer r.observe("notes_create", start)

	res, err := r.DB.ExecContext(ctx, `INSERT INTO notes(user_id,title,body) VALUES(?,?,?)`, uid, title, body)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func (r *Notes) Get(ctx context.Context, uid, id int64) (Note, error) {
	start := time.Now()
	defer r.observe("notes_get", start)

	var n Note
	err := r.DB.QueryRowContext(ctx, `
		SELECT id,user_id,title,body,created_at,updated_at,deleted_at
		FROM notes WHERE id=? AND user_id=? AND deleted_at IS NULL`,
		id, uid).
		Scan(&n.ID, &n.UserID, &n.Title, &n.Body, &n.CreatedAt, &n.UpdatedAt, &n.DeletedAt)
	return n, err
}

func (r *Notes) Update(ctx context.Context, uid, id int64, title, body string) (Note, error) {
	start := time.Now()
	defer r.observe("notes_update", start)

	_, err := r.DB.ExecContext(ctx, `UPDATE notes SET title=?, body=? WHERE id=? AND user_id=? AND deleted_at IS NULL`,
		title, body, id, uid)
	if err != nil {
		return Note{}, err
	}
	return r.Get(ctx, uid, id)
}

func (r *Notes) Delete(ctx context.Context, uid, id int64) (Note, error) {
	start := time.Now()
	defer r.observe("notes_delete", start)

	n, err := r.Get(ctx, uid, id)
	if err != nil {
		return Note{}, err
	}
	_, err = r.DB.ExecContext(ctx, `UPDATE notes SET deleted_at=NOW() WHERE id=? AND user_id=? AND deleted_at IS NULL`,
		id, uid)
	return n, err
}
