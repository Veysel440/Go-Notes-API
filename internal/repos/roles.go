package repos

import (
	"context"
	"database/sql"
)

type Roles struct{ DB *sql.DB }

func (r Roles) Assign(ctx context.Context, uid int64, role string) error {
	var rid int64
	if err := r.DB.QueryRowContext(ctx, `SELECT id FROM roles WHERE name=?`, role).Scan(&rid); err != nil {
		return err
	}
	_, err := r.DB.ExecContext(ctx, `INSERT IGNORE INTO user_roles(user_id,role_id) VALUES(?,?)`, uid, rid)
	return err
}

func (r Roles) Unassign(ctx context.Context, uid int64, role string) error {
	var rid int64
	if err := r.DB.QueryRowContext(ctx, `SELECT id FROM roles WHERE name=?`, role).Scan(&rid); err != nil {
		return err
	}
	_, err := r.DB.ExecContext(ctx, `DELETE FROM user_roles WHERE user_id=? AND role_id=?`, uid, rid)
	return err
}

func (r Roles) Has(ctx context.Context, uid int64, role string) (bool, error) {
	var n int
	err := r.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM user_roles ur
		JOIN roles ro ON ro.id=ur.role_id
		WHERE ur.user_id=? AND ro.name=?`, uid, role).Scan(&n)
	return n > 0, err
}
