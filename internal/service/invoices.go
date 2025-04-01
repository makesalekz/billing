package service

import (
	"context"

	v1 "gitlab.calendaria.team/services/finance/billing/api/billing/v1"
	"gitlab.calendaria.team/services/finance/billing/ent/enum"
	"gitlab.calendaria.team/services/finance/billing/internal/biz"
	"gitlab.calendaria.team/services/finance/billing/internal/data"
	"gitlab.calendaria.team/services/utils/v2/auth"
)

type InvoiceService struct {
	v1.UnimplementedInvoicesServer

	uc *biz.InvoicesUseCase
}

func NewInvoiceService(uc *biz.InvoicesUseCase) *InvoiceService {
	return &InvoiceService{uc: uc}
}

func (s *InvoiceService) GetInvoice(ctx context.Context, req *v1.GetInvoiceRequest) (*v1.InvoiceReply, error) {
	actorID := auth.GetActorIdFromContext(ctx)
	if actorID == 0 {
		return nil, v1.ErrorEmptyActorId("empty actor id")
	}

	tenantID := auth.GetTenantIdFromContext(ctx)
	if tenantID == 0 {
		return nil, v1.ErrorEmptyTenantId("empty tenant id")
	}

	appID := auth.GetAppIdFromContext(ctx)
	if appID == "" {
		return nil, v1.ErrorEmptyAppId("empty app id")
	}

	invoice, err := s.uc.GetInvoice(ctx, actorID, tenantID, appID, req.GetId())
	if err != nil {
		return nil, err
	}

	return &v1.InvoiceReply{Invoice: biz.ReplyInvoice(invoice)}, nil
}

func (s *InvoiceService) ListInvoices(ctx context.Context, req *v1.ListInvoicesRequest) (*v1.ListInvoicesReply, error) {
	actorID := auth.GetActorIdFromContext(ctx)
	if actorID == 0 {
		return nil, v1.ErrorEmptyActorId("empty actor id")
	}
	tenantID := auth.GetTenantIdFromContext(ctx)
	if tenantID == 0 {
		return nil, v1.ErrorEmptyTenantId("empty tenant id")
	}

	appID := auth.GetAppIdFromContext(ctx)
	if appID == "" {
		return nil, v1.ErrorEmptyAppId("empty app id")
	}

	status := enum.InvoiceStatus(req.GetStatus())
	if status != "" && !status.IsValid() {
		return nil, v1.ErrorInvalidRequest("invalid status")
	}

	filter := data.InvoiceFilter{
		ProductID:      req.GetProductId(),
		Status:         status,
		Paid:           req.GetPaid(),
		SubscriptionID: req.GetSubscriptionId(),
	}

	pagination := FormPaginateRequest(req.GetPagination())

	listInvoices, err := s.uc.ListInvoices(ctx, filter, pagination)
	if err != nil {
		return nil, err
	}

	return &v1.ListInvoicesReply{
		Invoices:   biz.ReplyInvoices(listInvoices.Invoices),
		Pagination: listInvoices.PaginateReply,
	}, nil
}
