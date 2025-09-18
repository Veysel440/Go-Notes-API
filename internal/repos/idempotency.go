package repos

import (
	"context"
	"database/sql"
	"errors"
)

type Idempotency struct{ DB *sql.DB }

var (
	ErrInProgress = errors.New("in_progress")
	ErrMismatch   = errors.New("body_mismatch")
)

func (r Idempotency) Claim(ctx context.Context, key string, uid int64, method, path, bodyHash string) (*int64, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	_, insErr := tx.ExecContext(ctx,
		"INSERT INTO idempotency_keys(`key`,user_id,method,path,body_sha256,note_id,claimed_at,completed_at) VALUES(?,?,?,?,?,NULL,NOW(),NULL)",
		key, uid, method, path, bodyHash,
	)
	if insErr == nil {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return nil, nil
	}

	var existing struct {
		Body string
		Note sql.NullInt64
		Comp sql.NullTime
	}
	row := tx.QueryRowContext(ctx,
		"SELECT body_sha256, note_id, completed_at FROM idempotency_keys WHERE `key`=? AND user_id=? FOR UPDATE",
		key, uid,
	)
	if err := row.Scan(&existing.Body, &existing.Note, &existing.Comp); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if existing.Body != bodyHash {
		_ = tx.Rollback()
		return nil, ErrMismatch
	}
	if !existing.Comp.Valid {
		_ = tx.Rollback()
		return nil, ErrInProgress
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	if existing.Note.Valid {
		id := existing.Note.Int64
		return &id, nil
	}
	return nil, ErrInProgress
}

func (r Idempotency) Complete(ctx context.Context, key string, uid, noteID int64) error {
	_, err := r.DB.ExecContext(ctx,
		"UPDATE idempotency_keys SET note_id=?, completed_at=NOW() WHERE `key`=? AND user_id=? AND completed_at IS NULL",
		noteID, key, uid,
	)
	return err
}
