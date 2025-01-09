package biz

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"golang.org/x/exp/maps"

	"gitlab.calendaria.team/services/finance/billing/messages"

	"github.com/go-kratos/kratos/v2/log"

	v1 "gitlab.calendaria.team/services/finance/billing/api/billing/v1"
	"gitlab.calendaria.team/services/finance/billing/ent"
	"gitlab.calendaria.team/services/finance/billing/ent/enum"
	"gitlab.calendaria.team/services/finance/billing/internal/data"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
	u_nats "gitlab.calendaria.team/services/utils/v2/nats"
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
		err = uc.canProductBeUsedOneMoreTime(ctx, actorID, product)
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

	if invoiceDto.Price.IsZero() {
		invoiceDto.Price = product.Price.Mul(decimal.NewFromInt(invoiceDto.Amount))
	}

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

	invoice, err := uc.invoiceRepo.UpdateInvoice(ctx, invoiceData, dto)
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
func (uc *InvoicesUseCase) canProductBeUsedOneMoreTime(ctx context.Context, actorID int64, product *ent.Product) error {
	if product.IsUnique {
		count, err := uc.invoiceRepo.CountInvoices(
			ctx, data.InvoiceFilter{
				UserID:    actorID,
				ProductID: product.ID,
				Status:    enum.Paid,
			},
		)
		if err != nil {
			return v1.ErrorDatabaseQuery("failed to list invoices: %s", err.Error())
		}

		if int64(count) >= product.UniqueLimit {
			return v1.ErrorInvalidRequest("product already used")
		}
	}

	return nil
}

func (uc *InvoicesUseCase) RevokeInvoice(
	ctx context.Context, actorID, tenantID int64, appID string, invoiceID int64,
) error {
	invoiceData, err := uc.invoiceRepo.GetInvoice(ctx, actorID, tenantID, appID, invoiceID)
	if err != nil {
		return v1.ErrorDatabaseQuery("failed to revoke invoice: %s", err.Error())
	}

	if invoiceData.Status != enum.Paid {
		return v1.ErrorInvalidRequest("invoice is not paid")
	}

	now := time.Now()
	tvar := true

	_, err = uc.invoiceRepo.UpdateInvoice(
		ctx, invoiceData, data.InvoiceDto{
			IsRevoked: &tvar,
			RevokedAt: &now,
		},
	)
	if err != nil {
		return v1.ErrorDatabaseQuery("failed to revoke invoice: %s", err.Error())
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

func (uc *InvoicesUseCase) UpdateResources(ctx context.Context) {
	fvar := false
	now := time.Now()
	invoices, err := uc.invoiceRepo.ListInvoices(
		ctx, data.InvoiceFilter{
			Status:        enum.Paid,
			PaidProcesses: &fvar,
			IsRevoked:     &fvar,
			PaidTillProc:  &fvar,
			PaidTill:      &now,
		}, &utils_v1.PaginateRequest{Limit: data.BackgroundProcessPageSize, FromId: 0},
	)
	if err != nil {
		uc.log.Errorf("failed to list invoices: %s", err.Error())

		return
	}

	for _, invoice := range invoices {
		err = uc.updateResources(ctx, invoice, invoice.ProductID)
		if err != nil {
			uc.log.Errorf("failed to update resources of invoice: %d, err %s", invoice.ID, err.Error())

			continue
		}
	}
}

func (uc *InvoicesUseCase) updateResources(ctx context.Context, invoice *ent.Invoice, productID int64) error {
	product, err := uc.productRepo.GetProduct(ctx, productID)
	if err != nil {
		return v1.ErrorInternal("failed to get product: %s", err.Error())
	}

	if product.Edges.Bundles == nil {
		return nil
	}

	bundles := product.Edges.Bundles

	itemIDs := make(map[int64]float64, len(bundles))
	for _, bundle := range bundles {
		itemIDs[bundle.ItemID] = bundle.Amount
	}

	items, err := uc.itemsRepo.GetItems(ctx, maps.Keys(itemIDs))
	if err != nil {
		return v1.ErrorInternal("failed to get items: %s", err.Error())
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
			IsTrial:  invoice.IsTrial,
		}

		uc.queryManager.GetLocal(*item.TopicName).Pub(refreshedItem)
	}

	tvar := true
	_, err = uc.invoiceRepo.UpdateInvoice(
		ctx, invoice, data.InvoiceDto{
			IsPaidAtProcessed: &tvar,
		},
	)
	if err != nil {
		return v1.ErrorInternal("failed to update invoice: %s", err.Error())
	}

	return nil
}

func (uc *InvoicesUseCase) ExpireResources(ctx context.Context) {
	now := time.Now().Add(time.Hour) // give one hour to renew subscription

	invoices, err := uc.invoiceRepo.GetInvoicesToExpire(ctx, &now)
	if err != nil {
		uc.log.Errorf("failed to list invoices: %s", err.Error())
	}

	for _, invoice := range invoices {
		err = uc.revokeResources(ctx, invoice, invoice.ProductID, true)
		if err != nil {
			uc.log.Errorf("failed to revoke resources of invoice: %d, err %s", invoice.ID, err.Error())
		}
	}
}

func (uc *InvoicesUseCase) RevokeResources(ctx context.Context) {
	now := time.Now()

	invoices, err := uc.invoiceRepo.GetInvoicesToRevoke(ctx, &now)
	if err != nil {
		uc.log.Errorf("failed to list invoices: %s", err.Error())

		return
	}

	for _, invoice := range invoices {
		err = uc.revokeResources(ctx, invoice, invoice.ProductID, false)
		if err != nil {
			uc.log.Errorf("failed to revoke resources of invoice: %d, err %s", invoice.ID, err.Error())

			continue
		}
	}
}

func (uc *InvoicesUseCase) revokeResources(
	ctx context.Context, invoice *ent.Invoice, productID int64, isExpired bool,
) error {
	product, err := uc.productRepo.GetProduct(ctx, productID)
	if err != nil {
		return v1.ErrorInternal("failed to get product: %s", err.Error())
	}

	if product.Edges.Bundles == nil {
		return nil
	}

	bundles := product.Edges.Bundles

	itemIDs := make(map[int64]float64, len(bundles))
	for _, bundle := range bundles {
		itemIDs[bundle.ItemID] = bundle.Amount
	}

	items, err := uc.itemsRepo.GetItems(ctx, maps.Keys(itemIDs))
	if err != nil {
		return v1.ErrorInternal("failed to get items: %s", err.Error())
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
			IsTrial:  invoice.IsTrial,
		}

		uc.queryManager.GetLocal(*item.TopicName + "_revoke").Pub(refreshedItem)
	}

	tvar := true
	dto := data.InvoiceDto{}

	if isExpired {
		dto.IsPaidTillProcessed = &tvar
	} else {
		dto.IsRevokedProcessed = &tvar
	}

	_, err = uc.invoiceRepo.UpdateInvoice(ctx, invoice, dto)
	if err != nil {
		return v1.ErrorInternal("failed to update invoice: %s", err.Error())
	}

	return nil
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
		Revoked:             invoice.IsRevoked,
		IsRevokedProcessed:  invoice.IsRevokedProcessed,
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

	if invoice.RevokedAt != nil {
		reply.RevokedAt = invoice.RevokedAt.Format(time.RFC3339)
	}

	if invoice.AppleStoreTransactionID != nil {
		reply.AppleStoreTransactionId = *invoice.AppleStoreTransactionID
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
