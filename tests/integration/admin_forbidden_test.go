//go:build integration

package integration

import (
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestAdmin_ForbiddenForUser(t *testing.T) {
	if err := waitReady(baseURL(), 30*time.Second); err != nil {
		t.Skip("api not ready:", err)
	}
	email := "adm_" + strconv.FormatInt(time.Now().UnixNano(), 10) + "@t.io"
	_, _ = register(email, "Password1!")
	acc, _, _ := login(email, "Password1!")
	h := authHeader(acc)

	req, _ := http.NewRequest(http.MethodGet, baseURL()+"/admin/users", nil)
	req.Header = h
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 403/401, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}
