package middleware

import (
	"database/sql"
	"net"
	"net/http"
)

type Audit struct{ DB *sql.DB }

type sw struct {
	http.ResponseWriter
	status int
}

func (w *sw) WriteHeader(c int) { w.status = c; w.ResponseWriter.WriteHeader(c) }

func (a Audit) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wr := &sw{ResponseWriter: w, status: 200}
		next.ServeHTTP(wr, r)
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		uid, _ := UserID(r.Context())
		_, _ = a.DB.ExecContext(r.Context(),
			`INSERT INTO audit_logs(user_id,method,path,status,ip,rid) VALUES(?,?,?,?,?,?)`,
			uid, r.Method, r.URL.Path, wr.status, host, r.Header.Get("X-Request-ID"),
		)
	})
}
