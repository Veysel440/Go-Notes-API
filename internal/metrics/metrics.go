package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Registry struct {
	reg     *prometheus.Registry
	HttpDur *prometheus.HistogramVec
	DbErr   *prometheus.CounterVec
}

func New() *Registry {
	r := prometheus.NewRegistry()
	httpDur := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"route", "method", "status"},
	)
	dbErr := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_errors_total",
			Help: "DB errors by operation",
		},
		[]string{"op"},
	)

	r.MustRegister(
		httpDur, dbErr,
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	)

	return &Registry{reg: r, HttpDur: httpDur, DbErr: dbErr}
}

func (r *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(r.reg, promhttp.HandlerOpts{})
}

func (r *Registry) MW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		ww := &wrap{ResponseWriter: w, status: 200}
		next.ServeHTTP(ww, req)
		r.HttpDur.WithLabelValues(req.URL.Path, req.Method, strconv.Itoa(ww.status)).
			Observe(time.Since(start).Seconds())
	})
}

func (r *Registry) Reg() *prometheus.Registry { return r.reg }

type wrap struct {
	http.ResponseWriter
	status int
}

func (w *wrap) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
