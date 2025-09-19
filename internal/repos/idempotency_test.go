package repos_test

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Veysel440/go-notes-api/internal/repos"
)

func TestIdem_Claim_InsertOK(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	idem := repos.Idem{DB: db}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO idempotency_keys")).
		WithArgs("k1", int64(7), "PUT", "/notes/1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	res, err := idem.Claim(context.Background(), "k1", 7, "PUT", "/notes/1", "h")
	if err != nil || res != nil {
		t.Fatalf("unexpected: res=%v err=%v", res, err)
	}
}

func TestIdem_Claim_ConflictCompleted(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	idem := repos.Idem{DB: db}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO idempotency_keys")).
		WillReturnError(sql.ErrConnDone)

	rows := sqlmock.NewRows([]string{"body_sha256", "completed_at", "result_text"}).
		AddRow("h", time.Now(), `{"ok":true}`)
	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT body_sha256, completed_at, result_text FROM idempotency_keys")).
		WithArgs("k1", int64(7)).WillReturnRows(rows)
	mock.ExpectCommit()

	res, err := idem.Claim(context.Background(), "k1", 7, "PUT", "/n", "h")
	if err != nil || res == nil {
		t.Fatalf("want completed result, got err=%v res=%v", err, res)
	}
}
