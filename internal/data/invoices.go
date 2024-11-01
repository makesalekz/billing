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
	GetInvoice(ctx context.Context, actorID int64, invoiceID int64) (*ent.Invoice, error)
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
		SetUserID(dto.ActorID).
		SetAppID(dto.AppID).
		SetProductID(dto.ProductID).
		SetStatus(dto.Status).
		SetPrice(dto.Price).
		SetAmount(dto.Amount)

	if dto.PaidAt != nil {
		query = query.SetPaidAt(*dto.PaidAt)
	}

	return query.Save(ctx)
}

func (r *invoicesRepo) UpdateInvoice(
	ctx context.Context, actorID, invoiceID int64, dto *InvoiceDto,
) (*ent.Invoice, error) {
	query := r.db.Invoice.UpdateOneID(invoiceID).
		SetStatus(dto.Status)

	if dto.PaidAt != nil {
		query = query.SetPaidAt(*dto.PaidAt)
	}

	return query.Save(ctx)
}

func (r *invoicesRepo) DeleteInvoice(ctx context.Context, actorID, invoiceID int64) error {
	return r.db.Invoice.DeleteOneID(invoiceID).
		Exec(ctx)
}

func (r *invoicesRepo) GetInvoice(ctx context.Context, actorID, invoiceID int64) (*ent.Invoice, error) {
	return r.db.Invoice.Query().
		Where(invoice.ID(invoiceID)).
		Only(ctx)
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

	return query.All(ctx)
}
