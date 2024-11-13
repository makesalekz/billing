package data

import (
	"context"

	"gitlab.calendaria.team/services/finance/invoices/ent"
	"gitlab.calendaria.team/services/finance/invoices/ent/invoice"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
)

type InvoicesRepo interface {
	CreateInvoice(ctx context.Context, actorID int64, dto *InvoiceDto) (*ent.Invoice, error)
	UpdateInvoice(ctx context.Context, actorID, invoiceID int64, dto *InvoiceDto) (*ent.Invoice, error)
	DeleteInvoice(ctx context.Context, actorID, invoiceID int64) error
	GetInvoice(ctx context.Context, actorID, tenantID int64, appID string, invoiceID int64) (*ent.Invoice, error)
	CountInvoices(ctx context.Context, actorID int64, filter InvoiceFilter) (int32, error)
	ListInvoices(
		ctx context.Context, actorID int64, filter InvoiceFilter, paginate *utils_v1.PaginateRequest,
	) ([]*ent.Invoice, error)
}

type invoicesRepo struct {
	db *ent.Client
}

func NewInvoicesRepo(d *Data) InvoicesRepo {
	return &invoicesRepo{
		db: d.db,
	}
}

func (r *invoicesRepo) CreateInvoice(ctx context.Context, actorID int64, dto *InvoiceDto) (*ent.Invoice, error) {
	query := r.db.Invoice.Create().
		SetUserID(dto.UserID).
		SetAppID(dto.AppID).
		SetProductID(dto.ProductID).
		SetStatus(dto.Status).
		SetPrice(dto.Price).
		SetAmount(dto.Amount)

	if dto.SubscriptionID != 0 {
		query.SetSubscriptionsID(dto.SubscriptionID)
	}

	if dto.PaidAt != nil {
		query = query.SetPaidAt(*dto.PaidAt)
	}

	return query.Save(ctx)
}

func (r *invoicesRepo) UpdateInvoice(
	ctx context.Context, actorID, invoiceID int64, dto *InvoiceDto,
) (*ent.Invoice, error) {
	query := r.db.Invoice.UpdateOneID(invoiceID).Where(invoice.UserID(actorID))

	if dto.Status.IsValid() {
		query.SetStatus(dto.Status)
	}

	if dto.PaidAt != nil {
		query = query.SetPaidAt(*dto.PaidAt)
	}

	return query.Save(ctx)
}

func (r *invoicesRepo) DeleteInvoice(ctx context.Context, actorID, invoiceID int64) error {
	return r.db.Invoice.
		DeleteOneID(invoiceID).
		Exec(ctx)
}

func (r *invoicesRepo) GetInvoice(
	ctx context.Context, actorID, tenantID int64, appID string, invoiceID int64,
) (*ent.Invoice, error) {
	return r.db.Invoice.Query().
		Where(
			invoice.ID(invoiceID),
			invoice.UserID(actorID),
			invoice.TenantID(tenantID),
			invoice.AppID(appID),
		).
		Only(ctx)
}

func (r *invoicesRepo) CountInvoices(ctx context.Context, actorID int64, filter InvoiceFilter) (int32, error) {
	query := r.db.Invoice.Query()

	if filter.Status.IsValid() {
		query.Where(invoice.StatusEQ(filter.Status))
	}

	if filter.ProductID != 0 {
		query.Where(invoice.ProductIDEQ(filter.ProductID))
	}

	if filter.Paid {
		query.Where(invoice.PaidAtNotNil())
	}

	if filter.SubscriptionID != 0 {
		query.Where(invoice.SubscriptionIDEQ(filter.SubscriptionID))
	}

	n, err := query.Count(ctx)
	if err != nil {
		return 0, err
	}

	return int32(n), err
}

func (r *invoicesRepo) ListInvoices(
	ctx context.Context, actorID int64, filter InvoiceFilter, paginate *utils_v1.PaginateRequest,
) ([]*ent.Invoice, error) {
	query := r.db.Invoice.Query().Where(invoice.IDGT(paginate.GetFromId())).Limit(int(paginate.Limit))

	if filter.Status.IsValid() {
		query.Where(invoice.StatusEQ(filter.Status))
	}

	if filter.ProductID != 0 {
		query.Where(invoice.ProductIDEQ(filter.ProductID))
	}

	if filter.Paid {
		query.Where(invoice.PaidAtNotNil())
	}

	if filter.SubscriptionID != 0 {
		query.Where(invoice.SubscriptionIDEQ(filter.SubscriptionID))
	}

	return query.All(ctx)
}
