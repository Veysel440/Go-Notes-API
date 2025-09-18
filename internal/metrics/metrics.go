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
	DbDur   *prometheus.HistogramVec
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

	dbDur := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "DB query latency",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"op"},
	)

	dbErr := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_errors_total",
			Help: "DB errors by op",
		},
		[]string{"op"},
	)

	r.MustRegister(
		httpDur, dbDur, dbErr,
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	)

	return &Registry{reg: r, HttpDur: httpDur, DbDur: dbDur, DbErr: dbErr}
}

func (r *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(r.reg, promhttp.HandlerOpts{})
}

func (r *Registry) MW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		ww := &statusWrap{ResponseWriter: w, status: 200}
		next.ServeHTTP(ww, req)
		r.HttpDur.WithLabelValues(req.URL.Path, req.Method, strconv.Itoa(ww.status)).
			Observe(time.Since(start).Seconds())
	})
}

func (r *Registry) Reg() *prometheus.Registry { return r.reg }

func (r *Registry) ObserveDB(op string, d time.Duration) {
	r.DbDur.WithLabelValues(op).Observe(d.Seconds())
}

type statusWrap struct {
	http.ResponseWriter
	status int
}

func (w *statusWrap) WriteHeader(code int) { w.status = code; w.ResponseWriter.WriteHeader(code) }
