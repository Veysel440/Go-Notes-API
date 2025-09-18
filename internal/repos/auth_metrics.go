package repos

import "github.com/prometheus/client_golang/prometheus"

type AuthMetrics struct{ Failed prometheus.Counter }

func NewAuthMetrics(reg *prometheus.Registry) *AuthMetrics {
	c := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "auth_failed_total", Help: "failed login attempts",
	})
	reg.MustRegister(c)
	return &AuthMetrics{Failed: c}
}
