package security

import (
	"crypto/rand"
	"encoding/binary"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/net/context"
)

type Brute struct {
	RDB    *redis.Client
	Limit  int
	Window time.Duration
}

func ipOf(r *http.Request) string {
	if h := r.Header.Get("X-Real-IP"); h != "" {
		return h
	}
	if h := r.Header.Get("X-Forwarded-For"); h != "" {
		return strings.Split(h, ",")[0]
	}
	host, _, _ := strings.Cut(r.RemoteAddr, ":")
	return host
}

func (b Brute) Key(r *http.Request, email string) string {
	return "brute:" + ipOf(r) + ":" + strings.ToLower(strings.TrimSpace(email))
}

func (b Brute) Allow(ctx context.Context, r *http.Request, email string) (ok bool, remaining int, ttl time.Duration) {
	key := b.Key(r, email)
	pipe := b.RDB.TxPipeline()
	cnt := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, b.Window)
	_, _ = pipe.Exec(ctx)
	v := int(cnt.Val())
	if v <= b.Limit {
		return true, b.Limit - v, 0
	}
	ttl, _ = b.RDB.TTL(ctx, key).Result()
	return false, 0, ttl
}

func JitterBackoff(base time.Duration) time.Duration {
	var b [8]byte
	_, _ = rand.Read(b[:])

	jit := time.Duration(binary.LittleEndian.Uint64(b[:])%250) * time.Millisecond
	return base + jit
}
