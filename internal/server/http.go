package server

import (
	v1 "gitlab.calendaria.team/services/dummy/api/dummy/v1"
	"gitlab.calendaria.team/services/dummy/internal/conf"
	"gitlab.calendaria.team/services/dummy/internal/service"
	"gitlab.calendaria.team/services/utils/v1/jwt"
	u_auth "gitlab.calendaria.team/services/utils/v1/middlewares/auth"
	"gitlab.calendaria.team/services/utils/v1/middlewares/metrics"

	prom "github.com/go-kratos/kratos/contrib/metrics/prometheus/v2"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var _metricSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "server",
	Subsystem: "requests",
	Name:      "duration_sec",
	Help:      "server requests duratio(sec).",
	Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.250, 0.5, 1},
}, []string{"kind", "operation"})

var _metricRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: "server",
	Subsystem: "requests",
	Name:      "code_total",
	Help:      "The total number of processed requests",
}, []string{"kind", "operation", "code", "reason"})

var _activeRequests = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: "server",
	Subsystem: "requests",
	Name:      "active_requests",
	Help:      "The total number of active requests",
}, []string{"kind", "operation"})

// NewHTTPServer new an HTTP server.
func NewHTTPServer(
	c *conf.Bootstrap,
	jwtp *jwt.JwtProcessor,
	dummy *service.DummyService,
) *khttp.Server {
	prometheus.MustRegister(_metricSeconds, _metricRequests, _activeRequests)

	var opts = []khttp.ServerOption{
		khttp.Middleware(
			recovery.Recovery(),
			metadata.Server(),
			u_auth.Server(jwtp),
			metrics.Server(
				metrics.WithSeconds(prom.NewHistogram(_metricSeconds)),
				metrics.WithRequests(prom.NewCounter(_metricRequests)),
				metrics.WithGauge(prom.NewGauge(_activeRequests)),
			),
		),
	}
	if c.Server.Http.Network != "" {
		opts = append(opts, khttp.Network(c.Server.Http.Network))
	}
	if c.Server.Http.Addr != "" {
		opts = append(opts, khttp.Address(c.Server.Http.Addr))
	}
	if c.Server.Http.Timeout != nil {
		opts = append(opts, khttp.Timeout(c.Server.Http.Timeout.AsDuration()))
	}
	srv := khttp.NewServer(opts...)

	v1.RegisterDummyHTTPServer(srv, dummy)

	srv.Handle("/metrics", promhttp.Handler())

	return srv
}
