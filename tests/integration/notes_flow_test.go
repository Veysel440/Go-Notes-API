//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestNotes_Flow_ETag_200(t *testing.T) {
	if err := waitReady(baseURL(), 30*time.Second); err != nil {
		t.Skip("api not ready:", err)
	}

	email := "nflow_" + strconv.FormatInt(time.Now().UnixNano(), 10) + "@t.io"
	_, _ = register(email, "Password1!")

	acc, _, err := login(email, "Password1!")
	if err != nil || acc == "" {
		t.Fatalf("login failed: %v", err)
	}
	h := authHeader(acc)

	req, _ := http.NewRequest(http.MethodPost, baseURL()+"/notes", mustJSON(map[string]string{
		"title": "t1", "body": "b1",
	}))
	req.Header = h
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("create err %v code %d", err, resp.StatusCode)
	}
	var cr struct {
		ID int64 `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&cr)
	resp.Body.Close()

	req, _ = http.NewRequest(http.MethodGet, baseURL()+"/notes/"+strconv.FormatInt(cr.ID, 10), nil)
	req.Header = h
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != 200 {
		t.Fatalf("get %d", resp.StatusCode)
	}
	etag := resp.Header.Get("ETag")
	if etag == "" {
		t.Fatal("missing ETag")
	}
	resp.Body.Close()

	req, _ = http.NewRequest(http.MethodGet, baseURL()+"/notes/"+strconv.FormatInt(cr.ID, 10), nil)
	req.Header = h
	req.Header.Set("If-None-Match", etag)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != 304 {
		t.Fatalf("if-none-match want 304, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	req, _ = http.NewRequest(http.MethodPut, baseURL()+"/notes/"+strconv.FormatInt(cr.ID, 10),
		mustJSON(map[string]string{"title": "t2", "body": "b2"}))
	req.Header = h
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != 200 {
		t.Fatalf("update want 200, got %d", resp.StatusCode)
	}
	var up map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&up)
	resp.Body.Close()
	if up["title"] != "t2" {
		t.Fatalf("update body not returned")
	}

	req, _ = http.NewRequest(http.MethodDelete, baseURL()+"/notes/"+strconv.FormatInt(cr.ID, 10), nil)
	req.Header = h
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != 200 {
		t.Fatalf("delete want 200, got %d", resp.StatusCode)
	}
	var del map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&del)
	resp.Body.Close()
}
