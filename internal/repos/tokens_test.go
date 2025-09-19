package repos_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Veysel440/go-notes-api/internal/repos"
)

func TestRefreshTokens_IssueUseRotate(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	r := &repos.RefreshTokens{DB: db}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO refresh_tokens")).
		WithArgs(int64(1), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if _, err := r.Issue(context.Background(), 1, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	rows := sqlmock.NewRows([]string{"user_id"}).AddRow(int64(1))
	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT user_id FROM refresh_tokens WHERE token=? AND revoked_at IS NULL")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(rows)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE refresh_tokens SET revoked_at")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO refresh_tokens")).
		WithArgs(int64(1), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(2, 1))

	_, _, reused, err := r.UseAndRotate(context.Background(), "tok", time.Now().Add(time.Hour))
	if err != nil || reused {
		t.Fatalf("unexpected: reused=%v err=%v", reused, err)
	}
}
