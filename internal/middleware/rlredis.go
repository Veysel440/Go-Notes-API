package middleware

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisLimiter struct {
	RDB       *redis.Client
	Limit     int
	Window    time.Duration
	KeyFn     func(*http.Request) string
	AllowCIDR string
	allowNet  *net.IPNet
}

func NewRedisLimiter(rdb *redis.Client, limit int, window time.Duration, keyFn func(*http.Request) string, allowCIDR string) *RedisLimiter {
	rl := &RedisLimiter{RDB: rdb, Limit: limit, Window: window, KeyFn: keyFn, AllowCIDR: allowCIDR}
	if allowCIDR != "" {
		if _, n, err := net.ParseCIDR(allowCIDR); err == nil {
			rl.allowNet = n
		}
	}
	return rl
}

func (l *RedisLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if l.allowNet != nil {
			ip := clientIP(r)
			if ip != nil && l.allowNet.Contains(ip) {
				next.ServeHTTP(w, r)
				return
			}
		}
		key := "rl:" + l.KeyFn(r)
		ctx := r.Context()
		pipe := l.RDB.TxPipeline()
		cnt := pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, l.Window)
		_, _ = pipe.Exec(ctx)
		if int(cnt.Val()) > l.Limit {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "rate_limited", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) net.IP {
	h := r.Header.Get("X-Real-IP")
	if h == "" {
		h = r.Header.Get("X-Forwarded-For")
		if h != "" {
			h = strings.Split(h, ",")[0]
		}
	}
	if h == "" {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		h = host
	}
	return net.ParseIP(h)
}
func IPKey(r *http.Request) string {
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
		if ip != "" {
			ip = strings.Split(ip, ",")[0]
		}
	}
	if ip == "" {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		ip = host
	}
	return "ip:" + ip
}
