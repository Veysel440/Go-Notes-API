package repos

import (
	"context"
	"database/sql"
	"time"
)

type User struct {
	ID           int64
	Email        string
	PasswordHash string
}

type Users struct{ DB *sql.DB }

func (r Users) Create(ctx context.Context, email, pass string) (int64, error) {
	res, err := r.DB.ExecContext(ctx, `insert into users(email,password_hash) values(?,?)`, email, pass)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func (r Users) FindByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := r.DB.QueryRowContext(ctx, `select id,email,password_hash from users where email=?`, email).
		Scan(&u.ID, &u.Email, &u.PasswordHash)
	return u, err
}

type UserRow struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
}

func (r Users) List(ctx context.Context, page, size int, q string) ([]UserRow, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	var (
		where string
		args  []any
	)
	if q != "" {
		where = " where email like ?"
		args = append(args, "%"+q+"%")
	}

	var total int64
	if err := r.DB.QueryRowContext(ctx, "select count(*) from users"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args2 := append(append([]any{}, args...), size, (page-1)*size)
	rows, err := r.DB.QueryContext(ctx,
		"select id,email from users"+where+" order by id desc limit ? offset ?", args2...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]UserRow, 0, size)
	for rows.Next() {
		var u UserRow
		if err := rows.Scan(&u.ID, &u.Email); err != nil {
			return nil, 0, err
		}
		out = append(out, u)
	}
	return out, total, rows.Err()
}

func withTimeout(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, d)
}
