package biz

import (
	"context"
	"errors"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/shopspring/decimal"

	v1 "gitlab.calendaria.team/services/finance/billing/api/billing/v1"
	"gitlab.calendaria.team/services/finance/billing/ent"
	"gitlab.calendaria.team/services/finance/billing/ent/enum"
	"gitlab.calendaria.team/services/finance/billing/internal/data"
)

type InvoicesManager struct {
	log                    *log.Helper
	productRepo            data.ProductRepo
	productReservationRepo data.ProductReservationRepo
	invoiceRepo            data.InvoicesRepo
}

func NewInvoicesManager(
	logger log.Logger,
	productRepo data.ProductRepo,
	productReservationRepo data.ProductReservationRepo,
	invoiceRepo data.InvoicesRepo,
) *InvoicesManager {
	return &InvoicesManager{
		log:                    log.NewHelper(log.With(logger, "module", "biz/invoices")),
		productRepo:            productRepo,
		productReservationRepo: productReservationRepo,
		invoiceRepo:            invoiceRepo,
	}
}

func (im *InvoicesManager) CreateInvoice(
	ctx context.Context, tenantID, actorID int64, invoiceDto data.InvoiceDto, productID int64,
) (*ent.Invoice, *ent.Product, func(), error) {
	if invoiceDto.Amount <= 0 {
		return nil, nil, nil, v1.ErrorInvalidRequest("amount must be greater than 0")
	}

	product, err := im.productRepo.GetProduct(ctx, productID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil, nil, v1.ErrorNotFound("product not found")
		}
		return nil, nil, nil, err
	}

	if !product.IsActive {
		return nil, nil, nil, errors.New("product is not active")
	}

	if err = im.canProductBeUsedOneMoreTime(ctx, actorID, product); err != nil {
		return nil, nil, nil, err
	}

	if err = im.checkProductLimit(invoiceDto.Amount, product); err != nil {
		return nil, nil, nil, err
	}

	if invoiceDto.Price.IsZero() || invoiceDto.Price.LessThanOrEqual(product.Price) {
		invoiceDto.Price = product.Price.Mul(decimal.NewFromInt(invoiceDto.Amount))
	}

	if product.PaymentModel == enum.Recurrent {
		hasActive, isFirst, err := im.checkSubscriptionStatus(ctx, tenantID, actorID, productID)
		if err != nil {
			return nil, nil, nil, v1.ErrorDatabaseQuery("%v", err)
		}

		if hasActive {
			return nil, nil, nil, v1.ErrorInvalidRequest("active subscription exists")
		}

		if isFirst {
			invoiceDto.Price = decimal.NewFromInt(DefaultPriceForCardLink)
			invoiceDto.IsTrial = true
		}
	}

	im.applyInvoicePeriod(product, invoiceDto)

	invoice, err := im.invoiceRepo.CreateInvoice(ctx, invoiceDto)
	if err != nil {
		return nil, nil, nil, v1.ErrorDatabaseQuery("failed to create invoice: %s", err)
	}

	rollbackFunc := func() {
		_, rollbackErr := im.invoiceRepo.UpdateInvoice(ctx, invoice, data.InvoiceDto{Status: enum.Failed})
		if rollbackErr != nil {
			im.log.Errorf("failed to rollback invoice: %v", rollbackErr)
		}

		err = im.productReservationRepo.CancelReservationByInvoiceID(ctx, invoice.ID)
		if err != nil {
			im.log.Errorf("failed to cancel product reservation: %v", err)
		}
	}

	_, err = im.productReservationRepo.CreateReservation(
		ctx, data.ProductReservationDto{
			ProductID:           invoiceDto.ProductID,
			InvoiceID:           invoice.ID,
			ReservationQuantity: invoiceDto.Amount,
			UserID:              actorID,
			Status:              enum.Pending,
		},
	)

	if err != nil {
		rollbackFunc()
		return nil, nil, nil, v1.ErrorDatabaseQuery("failed to create product reservation: %v", err)
	}

	return invoice, product, rollbackFunc, nil
}

func (im *InvoicesManager) applyInvoicePeriod(product *ent.Product, dto data.InvoiceDto) {
	if product.ProductPeriod == enum.ProductPeriodDay {
		paidTill := time.Now().AddDate(0, 0, 1)
		dto.PaidTill = &paidTill
	} else if product.ProductPeriod == enum.ProductPeriodWeek {
		paidTill := time.Now().AddDate(0, 0, 7)
		dto.PaidTill = &paidTill
	} else if product.ProductPeriod == enum.ProductPeriodMonth {
		paidTill := time.Now().AddDate(0, 1, 0)
		dto.PaidTill = &paidTill
	} else if product.ProductPeriod == enum.ProductPeriodYear {
		paidTill := time.Now().AddDate(1, 0, 0)
		dto.PaidTill = &paidTill
	} else if product.ProductPeriod == enum.ProductPeriodUnlimited {
		paidTill := time.Now().AddDate(100, 0, 0)
		dto.PaidTill = &paidTill
	}
}

func (im *InvoicesManager) checkSubscriptionStatus(
	ctx context.Context, tenantID, actorID, productID int64,
) (bool, bool, error) {
	// fetch all paid invoices for the user in the app
	filter := data.InvoiceFilter{
		TenantID:  tenantID,
		UserID:    actorID,
		Status:    enum.Paid,
		Paid:      true,
		ProductID: productID,
	}

	invoices, err := im.invoiceRepo.ListInvoices(ctx, filter, nil)
	if err != nil {
		return false, false, err
	}

	hasActive := false

	for _, invoice := range invoices {
		if invoice.PaidTill != nil && !invoice.IsRevoked {
			if invoice.PaidTill.After(time.Now()) {
				hasActive = true
			}
		}
	}

	isFirst := len(invoices) == 0

	return hasActive, isFirst, nil
}

func (im *InvoicesManager) canProductBeUsedOneMoreTime(ctx context.Context, actorID int64, product *ent.Product) error {
	if product.IsUnique {
		count, err := im.invoiceRepo.CountInvoices(
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

func (im *InvoicesManager) checkProductLimit(amount int64, product *ent.Product) error {
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
