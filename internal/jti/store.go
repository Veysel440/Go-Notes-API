package jti

import (
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/net/context"
)

type Store struct {
	RDB    *redis.Client
	Prefix string
}

func (s Store) IsRevoked(ctx context.Context, jti string) (bool, error) {
	ttl, err := s.RDB.TTL(ctx, s.Prefix+jti).Result()
	if err != nil {
		return false, err
	}
	return ttl > 0, nil
}

func (s Store) Revoke(ctx context.Context, jti string, ttl time.Duration) error {
	return s.RDB.SetEx(ctx, s.Prefix+jti, "1", ttl).Err()
}
