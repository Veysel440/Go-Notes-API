package repos

import (
	"context"
	"database/sql"
)

type Note struct {
	ID     int64  `json:"id"`
	UserID int64  `json:"-"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}
type Notes struct{ DB *sql.DB }

func (r Notes) List(ctx context.Context, uid int64) ([]Note, error) {
	rows, err := r.DB.QueryContext(ctx, `select id,user_id,title,body from notes where user_id=? order by id desc`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Note
	for rows.Next() {
		var n Note
		_ = rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Body)
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r Notes) Create(ctx context.Context, uid int64, title, body string) (int64, error) {
	res, err := r.DB.ExecContext(ctx, `insert into notes(user_id,title,body) values(?,?,?)`, uid, title, body)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func (r Notes) Get(ctx context.Context, uid, id int64) (Note, error) {
	var n Note
	err := r.DB.QueryRowContext(ctx, `select id,user_id,title,body from notes where id=? and user_id=?`, id, uid).
		Scan(&n.ID, &n.UserID, &n.Title, &n.Body)
	return n, err
}

func (r Notes) Update(ctx context.Context, uid, id int64, title, body string) error {
	_, err := r.DB.ExecContext(ctx, `update notes set title=?, body=? where id=? and user_id=?`, title, body, id, uid)
	return err
}

func (r Notes) Delete(ctx context.Context, uid, id int64) error {
	_, err := r.DB.ExecContext(ctx, `delete from notes where id=? and user_id=?`, id, uid)
	return err
}
