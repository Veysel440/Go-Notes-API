//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

func baseURL() string {
	if v := os.Getenv("BASE_URL"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

func waitReady(url string, timeout time.Duration) error {
	dead := time.Now().Add(timeout)
	for time.Now().Before(dead) {
		r, err := http.Get(url + "/readyz")
		if err == nil && r.StatusCode == 204 {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("ready timeout")
}

func mustJSON(v any) *bytes.Reader {
	b, _ := json.Marshal(v)
	return bytes.NewReader(b)
}

func register(email, pass string) (*http.Response, error) {
	return http.Post(baseURL()+"/auth/register", "application/json",
		mustJSON(map[string]string{"email": email, "password": pass}))
}

func login(email, pass string) (access, refresh string, err error) {
	r, err := http.Post(baseURL()+"/auth/login", "application/json",
		mustJSON(map[string]string{"email": email, "password": pass}))
	if err != nil {
		return "", "", err
	}
	defer r.Body.Close()
	var tok struct {
		Access  string `json:"access"`
		Refresh string `json:"refresh"`
	}
	_ = json.NewDecoder(r.Body).Decode(&tok)
	return tok.Access, tok.Refresh, nil
}

func authHeader(tok string) http.Header {
	h := http.Header{}
	h.Set("Authorization", "Bearer "+tok)
	h.Set("Content-Type", "application/json")
	return h
}
