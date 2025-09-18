package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env, Port                 string
	ReadTimeout, WriteTimeout time.Duration
	DBDsn                     string
	DBTimeout                 time.Duration
	JWTSecret                 []byte
	JWTTTL, RefreshTTL        time.Duration
	MaxBodyBytes              int64
	CorsOrigins               []string
	MetricsAllowCIDR          string
	BcryptCost                int
	RateRPS                   float64
	RateBurst                 int
}

// -------- helpers --------
func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
func mustDur(k, def string) time.Duration {
	v := getenv(k, def)
	d, err := time.ParseDuration(v)
	if err != nil {
		panic(k + ": invalid duration " + v)
	}
	return d
}
func mustInt(k, def string) int {
	v := getenv(k, def)
	n, err := strconv.Atoi(v)
	if err != nil {
		panic(k + ": invalid int " + v)
	}
	return n
}
func mustFloat(k, def string) float64 {
	v := getenv(k, def)
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		panic(k + ": invalid float " + v)
	}
	return f
}
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	a := strings.Split(s, ",")
	for i := range a {
		a[i] = strings.TrimSpace(a[i])
	}
	return a
}

// DB_DSN verilmemişse DB_* ile MySQL DSN üretir.
func mysqlDSNFromEnv() string {
	if dsn := os.Getenv("DB_DSN"); dsn != "" {
		return dsn
	}
	host := getenv("DB_HOST", "127.0.0.1")
	port := getenv("DB_PORT", "3306")
	name := getenv("DB_DATABASE", "notes")
	user := getenv("DB_USERNAME", "notes")
	pass := getenv("DB_PASSWORD", "notes")
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true&charset=utf8mb4",
		user, pass, host, port, name)
}

func Load() Config {
	return Config{
		Env:          getenv("APP_ENV", "dev"),
		Port:         getenv("APP_PORT", "8080"),
		ReadTimeout:  mustDur("APP_READ_TIMEOUT", "5s"),
		WriteTimeout: mustDur("APP_WRITE_TIMEOUT", "10s"),
		DBDsn:        mysqlDSNFromEnv(),
		DBTimeout:    mustDur("DB_TIMEOUT", "3s"),

		JWTSecret:  []byte(getenv("JWT_SECRET", "dev-secret")),
		JWTTTL:     mustDur("JWT_TTL", "15m"),
		RefreshTTL: mustDur("REFRESH_TTL", "720h"),

		MaxBodyBytes:     int64(mustInt("MAX_BODY_BYTES", "1048576")),
		CorsOrigins:      splitCSV(getenv("CORS_ORIGINS", "*")),
		MetricsAllowCIDR: getenv("METRICS_ALLOW", "127.0.0.1/32"),
		BcryptCost:       mustInt("BCRYPT_COST", "12"),
		RateRPS:          mustFloat("RATE_RPS", "10"),
		RateBurst:        mustInt("RATE_BURST", "10"),
	}
}
