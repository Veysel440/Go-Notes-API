package errors

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWrite_JSONShape(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	Write(rr, req, E(400, "bad_request", "bad", nil, nil))
	if rr.Code != 400 {
		t.Fatalf("code %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct == "" {
		t.Fatal("no content-type")
	}
	if rr.Body.Len() == 0 {
		t.Fatal("empty body")
	}
}
