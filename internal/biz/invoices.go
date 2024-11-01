package biz

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	v1 "gitlab.calendaria.team/services/finance/invoices/api/invoices/v1"
	"gitlab.calendaria.team/services/finance/invoices/ent"
	"gitlab.calendaria.team/services/finance/invoices/ent/enum"
	"gitlab.calendaria.team/services/finance/invoices/internal/data"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
)

type InvoicesUseCase struct {
	InvoiceRepo data.InvoicesRepo
	ItemsRepo   data.ItemsRepo
	ProductRepo data.ProductRepo
}

func NewInvoicesUseCase(
	invoiceRepo data.InvoicesRepo,
	itemsRepo data.ItemsRepo,
	productRepo data.ProductRepo,
) *InvoicesUseCase {
	return &InvoicesUseCase{
		InvoiceRepo: invoiceRepo,
		ItemsRepo:   itemsRepo,
		ProductRepo: productRepo,
	}
}

func (uc *InvoicesUseCase) CreateInvoice(
	ctx context.Context, actorID, tenantID int64, appID string, productID int64, amount int64,
) (*ent.Invoice, error) {
	if amount < 1 {
		return nil, v1.ErrorInvalidRequest("amount must be greater than 0")
	}

	invoiceDto := data.InvoiceDto{
		ActorID:   actorID,
		AppID:     appID,
		ProductID: productID,
		Amount:    amount,
		Status:    enum.Created,
	}

	product, err := uc.ProductRepo.GetProduct(ctx, productID)
	if err != nil {
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
		err = uc.checkProductLimit(amount, product)
		if err != nil {
			return nil, err
		}
	}

	invoiceDto.Amount = amount
	invoiceDto.Price = product.Price.Mul(decimal.NewFromInt(amount))

	invoice, err := uc.InvoiceRepo.CreateInvoice(ctx, actorID, &invoiceDto)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("failed to create invoice: %s", err.Error())
	}

	if invoiceDto.Price.IsZero() {
		go uc.proceedPayment(context.Background(), invoice.ID)
	}

	return invoice, nil
}

// checks if product was already used
func (uc *InvoicesUseCase) checkProductUniqueness(ctx context.Context, actorID int64, product *ent.Product) error {
	if product.IsUnique {
		invoices, err := uc.InvoiceRepo.ListInvoices(ctx, actorID, data.InvoiceFilter{
			ProductID: product.ID,
		}, &utils_v1.PaginateRequest{
			Limit:  100,
			FromId: 0,
		})
		if err != nil {
			return v1.ErrorDatabaseQuery("failed to list invoices: %s", err.Error())
		}

		if len(invoices) > 0 {
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

func (uc *InvoicesUseCase) proceedPayment(ctx context.Context, invoiceID int64) bool {

	return true
}
