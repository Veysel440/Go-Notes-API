package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type Limiter struct {
	r   rate.Limit
	b   int
	m   sync.Map // key -> *rate.Limiter
	ttl time.Duration
}

func NewLimiter(r rate.Limit, burst int, ttl time.Duration) *Limiter {
	return &Limiter{r: r, b: burst, ttl: ttl}
}

func (l *Limiter) get(k string) *rate.Limiter {
	now := time.Now()
	v, ok := l.m.Load(k)
	if ok {
		p := v.(*entry)
		p.ts = now
		return p.lim
	}
	lim := rate.NewLimiter(l.r, l.b)
	l.m.Store(k, &entry{lim: lim, ts: now})
	return lim
}

type entry struct {
	lim *rate.Limiter
	ts  time.Time
}

func (l *Limiter) Cleanup() {
	cut := time.Now().Add(-l.ttl)
	l.m.Range(func(key, value any) bool {
		if value.(*entry).ts.Before(cut) {
			l.m.Delete(key)
		}
		return true
	})
}

func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		if host == "" {
			host = r.RemoteAddr
		}
		if !l.get(host).Allow() {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "rate limit", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
