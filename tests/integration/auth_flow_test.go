//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"
)

func baseURL() string {
	if v := os.Getenv("BASE_URL"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

func TestAuthFlow(t *testing.T) {
	b := baseURL()
	body := bytes.NewBufferString(`{"email":"a@b.c","password":"Password1!"}`)
	resp, err := http.Post(b+"/auth/register", "application/json", body)
	if err != nil || (resp.StatusCode != 200 && resp.StatusCode != 409) {
		t.Fatalf("register err %v code %d", err, resp.StatusCode)
	}

	body = bytes.NewBufferString(`{"email":"a@b.c","password":"Password1!"}`)
	resp, err = http.Post(b+"/auth/login", "application/json", body)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("login err %v code %d", err, resp.StatusCode)
	}
	var tok struct{ Access, Refresh string }
	_ = json.NewDecoder(resp.Body).Decode(&tok)
	if tok.Access == "" || tok.Refresh == "" {
		t.Fatal("empty tokens")
	}
}
