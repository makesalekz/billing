package server

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/robfig/cron/v3"

	"gitlab.calendaria.team/services/finance/billing/internal/biz"
)

type CronServer struct {
	log  *log.Helper
	cron *cron.Cron
}

// NewCronServer
func NewCronServer(
	logger log.Logger,
	invoice *biz.InvoicesUseCase,
	payments *biz.PaymentUseCase,
) *CronServer {
	cs := &CronServer{
		log:  log.NewHelper(log.With(logger, "module", "server/cron")),
		cron: cron.New(),
	}

	cs.processInvoices(invoice, payments)

	return cs
}

func (cs *CronServer) processInvoices(uc *biz.InvoicesUseCase, pc *biz.PaymentUseCase) {
	entryId, err := cs.cron.AddFunc(
		"@every 1m", func() {
			uc.UpdateResources(context.Background())
			uc.RevokeResources(context.Background())
			uc.ExpireResources(context.Background())

			pc.ProcessExpiredPayments(context.Background())
		},
	)
	if err != nil {
		cs.log.Errorf("failed on cron entryId: %v, err: %v", entryId, err)
		return
	}
}

func (cs *CronServer) Start(ctx context.Context) error {
	cs.cron.Start()
	cs.log.Info("cron server started")

	return nil
}

func (cs *CronServer) Stop(ctx context.Context) error {
	cs.cron.Stop()
	cs.log.Info("cron server stopped")

	return nil
}
