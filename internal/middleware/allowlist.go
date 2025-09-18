package middleware

import (
	"net"
	"net/http"
)

func AllowCIDR(cidr string) func(http.Handler) http.Handler {
	_, netw, err := net.ParseCIDR(cidr)
	if err != nil {
		return func(h http.Handler) http.Handler { return h }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host, _, _ := net.SplitHostPort(r.RemoteAddr)
			ip := net.ParseIP(host)
			if ip == nil || !netw.Contains(ip) {
				http.Error(w, "forbidden", 403)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
