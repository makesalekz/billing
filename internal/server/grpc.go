package server

import (
	v1 "github.com/makesalekz/billing/api/billing/v1"
	"github.com/makesalekz/billing/internal/conf"
	"github.com/makesalekz/billing/internal/service"
	"github.com/makesalekz/utils/v1/middlewares/metrics"
	"github.com/makesalekz/utils/v2/jwt"
	"github.com/makesalekz/utils/v2/middlewares/auth"
	u_tracing "github.com/makesalekz/utils/v2/tracing"

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
	productService *service.ProductService,
	invoiceService *service.InvoiceService,
	subscriptionService *service.SubscriptionService,
	appleStoreService *service.AppleStoreService,
	paymentService *service.PaymentsService,
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
	if c.GetServer().GetGrpc().GetNetwork() != "" {
		opts = append(opts, grpc.Network(c.GetServer().GetGrpc().GetNetwork()))
	}
	if c.GetServer().GetGrpc().GetAddr() != "" {
		opts = append(opts, grpc.Address(c.GetServer().GetGrpc().GetAddr()))
	}
	if c.GetServer().GetGrpc().GetTimeout() != nil {
		opts = append(opts, grpc.Timeout(c.GetServer().GetGrpc().GetTimeout().AsDuration()))
	}
	srv := grpc.NewServer(opts...)

	v1.RegisterItemsServer(srv, itemService)
	v1.RegisterProductsServer(srv, productService)
	v1.RegisterInvoicesServer(srv, invoiceService)
	v1.RegisterSubscriptionsServer(srv, subscriptionService)
	v1.RegisterAppleStoreServer(srv, appleStoreService)
	v1.RegisterPaymentsServer(srv, paymentService)

	return srv
}
