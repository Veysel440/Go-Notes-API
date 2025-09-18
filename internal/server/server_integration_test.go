package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Veysel440/go-notes-api/internal/config"
	dbx "github.com/Veysel440/go-notes-api/internal/db"
	"github.com/Veysel440/go-notes-api/internal/repos"
)

func Test_AdminRoleAndAudit(t *testing.T) {
	dsn := os.Getenv("TEST_DSN")
	if dsn == "" {
		t.Skip("TEST_DSN yok; entegrasyon testi atlandı")
	}
	os.Setenv("DB_DSN", dsn)

	cfg := config.Load()
	db, closeFn, err := dbx.OpenAndMigrate(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { closeFn(); db.Close() }()

	s := New(cfg, db)
	ts := httptest.NewServer(s.router())
	defer ts.Close()

	body := []byte(`{"email":"admin@test.local","password":"P@ssw0rd!"}`)
	res, err := http.Post(ts.URL+"/auth/register", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var reg struct {
		ID int64 `json:"id"`
	}
	_ = json.NewDecoder(res.Body).Decode(&reg)
	if reg.ID == 0 {
		t.Fatal("kayıt başarısız")
	}

	if err := (repos.Roles{DB: db}).Assign(context.Background(), reg.ID, "admin"); err != nil {
		t.Fatal(err)
	}

	res2, err := http.Post(ts.URL+"/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res2.Body.Close()
	var tk struct{ Access, Refresh string }
	_ = json.NewDecoder(res2.Body).Decode(&tk)
	if tk.Access == "" {
		t.Fatal("login token yok")
	}

	h := http.Header{"Authorization": {"Bearer " + tk.Access}}

	rolePayload := []byte(`{"action":"add","role":"user"}`)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/admin/users/"+itoa(reg.ID)+"/roles", bytes.NewReader(rolePayload))
	req.Header = h
	req.Header.Set("Content-Type", "application/json")
	res3, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res3.StatusCode != 204 {
		t.Fatalf("rol ekle status=%d", res3.StatusCode)
	}

	req2, _ := http.NewRequest(http.MethodGet, ts.URL+"/admin/audit?limit=10", nil)
	req2.Header = h
	res4, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	if res4.StatusCode != 200 {
		t.Fatalf("audit status=%d", res4.StatusCode)
	}
}

func itoa(n int64) string { return fmt.Sprintf("%d", n) }
