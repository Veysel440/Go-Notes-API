package repos

import (
	"context"
	"database/sql"
)

type Idem struct{ DB *sql.DB }

func (r Idem) Claim(ctx context.Context, key string, uid int64, method, path, bodyHash string) (*string, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	_, insErr := tx.ExecContext(ctx,
		"INSERT INTO idempotency_keys(`key`,user_id,method,path,body_sha256,claimed_at,completed_at,result_text) VALUES(?,?,?,?,?,NOW(),NULL,NULL)",
		key, uid, method, path, bodyHash)
	if insErr == nil {
		_ = tx.Commit()
		return nil, nil
	}

	var body string
	var completed sql.NullTime
	var result sql.NullString
	row := tx.QueryRowContext(ctx, "SELECT body_sha256, completed_at, result_text FROM idempotency_keys WHERE `key`=? AND user_id=? FOR UPDATE", key, uid)
	if err := row.Scan(&body, &completed, &result); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if body != bodyHash {
		_ = tx.Rollback()
		return nil, ErrMismatch
	}
	if !completed.Valid {
		_ = tx.Rollback()
		return nil, ErrInProgress
	}
	_ = tx.Commit()
	if result.Valid {
		s := result.String
		return &s, nil
	}
	return nil, ErrInProgress
}

func (r Idem) Complete(ctx context.Context, key string, uid int64, result string) error {
	_, err := r.DB.ExecContext(ctx,
		"UPDATE idempotency_keys SET result_text=?, completed_at=NOW() WHERE `key`=? AND user_id=? AND completed_at IS NULL",
		result, key, uid)
	return err
}
