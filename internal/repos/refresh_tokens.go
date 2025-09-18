package repos

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"
)

type RefreshTokens struct{ DB *sql.DB }

func (r RefreshTokens) Issue(ctx context.Context, uid int64, exp time.Time) (string, error) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	tok := hex.EncodeToString(b)
	_, err := r.DB.ExecContext(ctx, `INSERT INTO refresh_tokens(token,user_id,expires_at,used_at) VALUES(?,?,?,NULL)`, tok, uid, exp)
	return tok, err
}

func (r RefreshTokens) Use(ctx context.Context, token string) (int64, bool, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, false, err
	}
	var uid int64
	var usedAt sql.NullTime
	if err := tx.QueryRowContext(ctx, `SELECT user_id, used_at FROM refresh_tokens WHERE token=? AND expires_at>NOW() FOR UPDATE`, token).Scan(&uid, &usedAt); err != nil {
		_ = tx.Rollback()
		return 0, false, err
	}
	reused := usedAt.Valid

	if _, err := tx.ExecContext(ctx, `UPDATE refresh_tokens SET used_at=NOW() WHERE token=?`, token); err != nil {
		_ = tx.Rollback()
		return 0, false, err
	}
	if reused {
	
		_, _ = tx.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE user_id=? AND expires_at>NOW()`, uid)
	}
	return uid, reused, tx.Commit()
}
