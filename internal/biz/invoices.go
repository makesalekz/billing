package biz

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"gitlab.calendaria.team/services/finance/billing/messages"
	"golang.org/x/exp/maps"

	"github.com/go-kratos/kratos/v2/log"
	v1 "gitlab.calendaria.team/services/finance/billing/api/billing/v1"
	"gitlab.calendaria.team/services/finance/billing/ent"
	"gitlab.calendaria.team/services/finance/billing/ent/enum"
	"gitlab.calendaria.team/services/finance/billing/internal/data"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
	u_nats "gitlab.calendaria.team/services/utils/v1/nats"
)

type InvoicesList struct {
	Invoices      []*ent.Invoice
	PaginateReply *utils_v1.PaginateReply
}

type InvoicesUseCase struct {
	log          *log.Helper
	invoiceRepo  data.InvoicesRepo
	itemsRepo    data.ItemsRepo
	productRepo  data.ProductRepo
	queryManager u_nats.IQueueManager
}

func NewInvoicesUseCase(
	logger log.Logger,
	invoiceRepo data.InvoicesRepo,
	itemsRepo data.ItemsRepo,
	productRepo data.ProductRepo,
	queryManager u_nats.IQueueManager,
) *InvoicesUseCase {
	return &InvoicesUseCase{
		log:          log.NewHelper(log.With(logger, "module", "biz/project")),
		invoiceRepo:  invoiceRepo,
		itemsRepo:    itemsRepo,
		productRepo:  productRepo,
		queryManager: queryManager,
	}
}

func (uc *InvoicesUseCase) CreateInvoice(
	ctx context.Context, actorID, tenantID int64, appID string, invoiceDto data.InvoiceDto,
) (*ent.Invoice, error) {
	if invoiceDto.Amount <= 0 {
		return nil, v1.ErrorInvalidRequest("amount must be greater than 0")
	}

	product, err := uc.productRepo.GetProduct(ctx, invoiceDto.ProductID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, v1.ErrorNotFound("failed to get product: %s", err.Error())
		}

		return nil, v1.ErrorDatabaseQuery("failed to get product: %s", err.Error())
	}

	if !product.IsActive {
		return nil, v1.ErrorInvalidRequest("product is not active")
	}

	if product.IsUnique {
		err = uc.checkProductUniqueness(ctx, actorID, product)
		if err != nil {
			return nil, err
		}
	}

	if product.IsLimited {
		err = uc.checkProductLimit(invoiceDto.Amount, product)
		if err != nil {
			return nil, err
		}
	}

	invoiceDto.Price = product.Price.Mul(decimal.NewFromInt(invoiceDto.Amount))

	invoice, err := uc.invoiceRepo.CreateInvoice(ctx, invoiceDto)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("failed to create invoice: %s", err.Error())
	}

	return invoice, nil
}

func (uc *InvoicesUseCase) UpdateInvoice(
	ctx context.Context, actorID, tenantID int64, appID string, invoiceID int64, dto data.InvoiceDto,
) (*ent.Invoice, error) {
	invoiceData, err := uc.invoiceRepo.GetInvoice(ctx, actorID, tenantID, appID, invoiceID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, v1.ErrorNotFound("no such invoice was found, err %s", err.Error())
		}

		return nil, v1.ErrorDatabaseQuery("failed to update invoice, err %s", err.Error())
	}

	if dto.Status == enum.Paid && invoiceData.Status != enum.Paid {
		uc.updateResources(ctx, invoiceData, invoiceData.ProductID)
	}

	invoice, err := uc.invoiceRepo.UpdateInvoice(ctx, actorID, invoiceID, dto)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("failed to update invoice, err %s", err.Error())
	}

	return invoice, nil
}

func (uc *InvoicesUseCase) GetInvoice(
	ctx context.Context, actorID, tenantID int64, appID string, invoiceID int64,
) (*ent.Invoice, error) {
	invoice, err := uc.invoiceRepo.GetInvoice(ctx, actorID, tenantID, appID, invoiceID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, v1.ErrorNotFound("no such invoice was found, err %s", err.Error())
		}
		return nil, v1.ErrorDatabaseQuery("failed to get invoice, err %s", err.Error())
	}

	return invoice, nil
}

func (uc *InvoicesUseCase) ListInvoices(
	ctx context.Context, actorID int64, filter data.InvoiceFilter, paginate *utils_v1.PaginateRequest,
) (*InvoicesList, error) {
	total, err := uc.invoiceRepo.CountInvoices(ctx, filter)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("failed to list invoices, err %s", err.Error())
	}

	invoices, err := uc.invoiceRepo.ListInvoices(ctx, filter, paginate)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("failed to list invoices, err %s", err.Error())
	}

	paginateReply := &utils_v1.PaginateReply{
		Total: &total,
	}

	if len(invoices) > 0 {
		paginateReply.FromId = &invoices[len(invoices)-1].ID
	}

	return &InvoicesList{
		Invoices:      invoices,
		PaginateReply: paginateReply,
	}, nil
}

// checks if product was already used.
func (uc *InvoicesUseCase) checkProductUniqueness(ctx context.Context, actorID int64, product *ent.Product) error {
	if product.IsUnique {
		invoices, err := uc.invoiceRepo.ListInvoices(ctx, data.InvoiceFilter{
			UserID:    actorID,
			ProductID: product.ID,
			Status:    enum.Paid,
		}, &utils_v1.PaginateRequest{
			Limit:  DefaultPageSize, // I have limited unique_limit to 100
			FromId: 0,
		})
		if err != nil {
			return v1.ErrorDatabaseQuery("failed to list invoices: %s", err.Error())
		}

		if int64(len(invoices)) > product.UniqueLimit {
			return v1.ErrorInvalidRequest("product already used")
		}
	}

	return nil
}

func (uc *InvoicesUseCase) checkProductLimit(amount int64, product *ent.Product) error {
	if product.IsLimited {
		if product.Left <= 0 || product.Left < amount {
			return v1.ErrorInvalidRequest("product is out of stock")
		}

		if product.LimitedTill != nil && !product.LimitedTill.IsZero() && product.LimitedTill.Before(time.Now()) {
			return v1.ErrorInvalidRequest("product is no longer available")
		}
	}

	return nil
}

func (uc *InvoicesUseCase) updateResources(ctx context.Context, invoice *ent.Invoice, productID int64) {
	product, err := uc.productRepo.GetProduct(ctx, productID)
	if err != nil {
		uc.log.Errorf("failed to update resources, err %s", err.Error())
	}

	bundles := product.Edges.Bundles

	itemIDs := make(map[int64]float64, len(bundles))
	for _, bundle := range bundles {
		itemIDs[bundle.ItemID] = bundle.Amount
	}

	items, err := uc.itemsRepo.GetItems(ctx, maps.Keys(itemIDs))
	if err != nil {
		uc.log.Errorf("failed to get items: %s", err.Error())

		return
	}

	for _, item := range items {
		if item.TopicName == nil {
			continue
		}

		refreshedItem := messages.RefreshItems{
			Item:     item,
			Amount:   itemIDs[item.ID] * float64(invoice.Amount),
			UserID:   invoice.UserID,
			TenantID: invoice.TenantID,
			AppID:    invoice.AppID,
		}

		uc.queryManager.GetRemote(*item.TopicName).Pub(refreshedItem)
	}
}

func ReplyInvoice(invoice *ent.Invoice) *v1.Invoice {
	reply := &v1.Invoice{
		Id:                  invoice.ID,
		UserId:              invoice.UserID,
		TenantId:            invoice.TenantID,
		AppId:               invoice.AppID,
		ProductId:           invoice.ProductID,
		Amount:              invoice.Amount,
		Price:               invoice.Price.String(),
		Currency:            invoice.Currency,
		IsPaidAtProcessed:   invoice.IsPaidAtProcessed,
		IsPaidTillProcessed: invoice.IsPaidTillProcessed,
	}

	if invoice.PaidAt != nil {
		reply.PaidAt = invoice.PaidAt.Format(time.RFC3339)
	}

	if invoice.PaidTill != nil {
		reply.PaidTill = invoice.PaidTill.Format(time.RFC3339)
	}

	if invoice.SubscriptionID != nil {
		reply.SubscriptionId = *invoice.SubscriptionID
	}

	return reply
}

func ReplyInvoices(invoices []*ent.Invoice) []*v1.Invoice {
	reply := make([]*v1.Invoice, len(invoices))

	for i, invoice := range invoices {
		reply[i] = ReplyInvoice(invoice)
	}

	return reply
}
