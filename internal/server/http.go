package server

import (
	"gitlab.calendaria.team/services/finance/billing/internal/conf"
	"gitlab.calendaria.team/services/utils/v1/middlewares/metrics"
	"gitlab.calendaria.team/services/utils/v4/jwt"
	"gitlab.calendaria.team/services/utils/v4/middlewares/auth"

	prom "github.com/go-kratos/kratos/contrib/metrics/prometheus/v2"
	"github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

//nolint:gochecknoglobals,promlinter // global variable, used in metrics
var _metricSeconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "server",
		Subsystem: "requests",
		Name:      "duration_sec",
		Help:      "server requests duratio(sec).",
		Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.250, 0.5, 1},
	}, []string{"kind", "operation"},
)

//nolint:gochecknoglobals // global variable, used in metrics
var _metricRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "server",
		Subsystem: "requests",
		Name:      "code_total",
		Help:      "The total number of processed requests",
	}, []string{"kind", "operation", "code", "reason"},
)

//nolint:gochecknoglobals // global variable, used in metrics
var _activeRequests = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: "server",
		Subsystem: "requests",
		Name:      "active_requests",
		Help:      "The total number of active requests",
	}, []string{"kind", "operation"},
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(
	c *conf.Bootstrap,
	jwtp jwt.IJwtProcessor,
) *khttp.Server {
	prometheus.MustRegister(_metricSeconds, _metricRequests, _activeRequests)

	var opts = []khttp.ServerOption{
		khttp.Middleware(
			recovery.Recovery(),
			metadata.Server(),
			auth.Server(jwtp),
			metrics.Server(
				metrics.WithSeconds(prom.NewHistogram(_metricSeconds)),
				metrics.WithRequests(prom.NewCounter(_metricRequests)),
				metrics.WithGauge(prom.NewGauge(_activeRequests)),
			),
		),
	}
	if c.GetServer().GetHttp().GetNetwork() != "" {
		opts = append(opts, khttp.Network(c.GetServer().GetHttp().GetNetwork()))
	}
	if c.GetServer().GetHttp().GetAddr() != "" {
		opts = append(opts, khttp.Address(c.GetServer().GetHttp().GetAddr()))
	}
	if c.GetServer().GetHttp().GetTimeout() != nil {
		opts = append(opts, khttp.Timeout(c.GetServer().GetHttp().GetTimeout().AsDuration()))
	}
	srv := khttp.NewServer(opts...)

	srv.Handle("/metrics", promhttp.Handler())

	return srv
}
