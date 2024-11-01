package server

import (
	v1 "gitlab.calendaria.team/services/finance/invoices/api/invoices/v1"
	// v1 "gitlab.calendaria.team/services/dummy/api/dummy/v1"
	"gitlab.calendaria.team/services/finance/invoices/internal/conf"
	"gitlab.calendaria.team/services/finance/invoices/internal/service"
	"gitlab.calendaria.team/services/utils/v1/middlewares/metrics"
	"gitlab.calendaria.team/services/utils/v2/jwt"
	"gitlab.calendaria.team/services/utils/v2/middlewares/auth"
	u_tracing "gitlab.calendaria.team/services/utils/v2/tracing"

	prom "github.com/go-kratos/kratos/contrib/metrics/prometheus/v2"
	"github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(
	c *conf.Bootstrap,
	jwtp jwt.IJwtProcessor,
	tracer *u_tracing.Tracer,
	itemService *service.ItemService,
) *grpc.Server {
	err := tracer.Initialize()
	if err != nil {
		panic(err)
	}

	var opts = []grpc.ServerOption{
		grpc.Middleware(
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
	if c.Server.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Server.Grpc.Network))
	}
	if c.Server.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Server.Grpc.Addr))
	}
	if c.Server.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Server.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)

	v1.RegisterItemsServer(srv, itemService)

	return srv
}
