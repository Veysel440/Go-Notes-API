package redisx

import (
	"crypto/tls"

	"github.com/Veysel440/go-notes-api/internal/config"
	"github.com/redis/go-redis/v9"
)

func New(cfg config.Config) *redis.Client {
	opts := &redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
		DB:       cfg.RedisDB,
	}
	if cfg.RedisTLS {
		opts.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	return redis.NewClient(opts)
}
