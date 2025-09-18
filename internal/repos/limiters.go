package repos

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

func NewEmailLimiter() func(string) bool {
	var m sync.Map
	return func(key string) bool {
		v, _ := m.LoadOrStore(key, rate.NewLimiter(rate.Every(2*time.Second), 3))
		return v.(*rate.Limiter).Allow()
	}
}
