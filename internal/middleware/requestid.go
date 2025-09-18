package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

const ReqIDHeader = "X-Request-ID"

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(ReqIDHeader)
		if id == "" {
			var b [16]byte
			_, _ = rand.Read(b[:])
			id = hex.EncodeToString(b[:])
		}
		w.Header().Set(ReqIDHeader, id)
		r.Header.Set(ReqIDHeader, id)
		next.ServeHTTP(w, r)
	})
}
