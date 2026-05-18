package service

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/makesalekz/billing/api/billing/v1"
	"github.com/makesalekz/billing/ent/enum"
	"github.com/makesalekz/billing/internal/biz"
	"github.com/makesalekz/billing/internal/data"
	"github.com/makesalekz/utils/v2/auth"
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

func (s *InvoiceService) GetInvoiceReceipt(ctx context.Context, req *v1.GetInvoiceReceiptRequest) (*v1.InvoiceReceiptReply, error) {
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

	invoice, err := s.uc.GetInvoice(ctx, actorID, tenantID, appID, req.GetInvoiceId())
	if err != nil {
		return nil, err
	}

	product, _ := s.uc.GetProduct(ctx, invoice.ProductID)

	reply := &v1.InvoiceReceiptReply{
		InvoiceId:  invoice.ID,
		Status:     string(invoice.Status),
		Quantity:   invoice.Amount,
		TotalPrice: invoice.Price.String(),
		Currency:   invoice.Currency,
		UserId:     invoice.UserID,
		TenantId:   invoice.TenantID,
	}

	if product != nil {
		reply.ProductName = product.Name
		reply.ProductDescription = product.Description
		reply.UnitPrice = product.Price.String()
	}

	if invoice.PaidAt != nil {
		reply.InvoiceDate = invoice.PaidAt.Format(time.RFC3339)
		reply.PaidAt = invoice.PaidAt.Format(time.RFC3339)
	}
	if invoice.PaidTill != nil {
		reply.PaidTill = invoice.PaidTill.Format(time.RFC3339)
	}
	if invoice.ExternalTransactionID != nil {
		reply.TransactionId = *invoice.ExternalTransactionID
	}

	// Card info from payment profile
	if invoice.Edges.PaymentProfile != nil {
		profile := invoice.Edges.PaymentProfile
		reply.CardLastFour = profile.PanMasked
	}

	return reply, nil
}

func (s *InvoiceService) GetInvoicePDF(ctx context.Context, req *v1.GetInvoiceReceiptRequest) (*v1.InvoicePDFReply, error) {
	receipt, err := s.GetInvoiceReceipt(ctx, req)
	if err != nil {
		return nil, err
	}

	pdfBytes, err := biz.GenerateInvoicePDF(receipt)
	if err != nil {
		return nil, v1.ErrorInternal("failed to generate PDF: %v", err)
	}

	filename := fmt.Sprintf("qalai-invoice-%d.pdf", req.GetInvoiceId())
	return &v1.InvoicePDFReply{
		PdfData:  pdfBytes,
		Filename: filename,
	}, nil
}
