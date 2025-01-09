package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"gitlab.calendaria.team/services/finance/billing/ent"
	"gitlab.calendaria.team/services/finance/billing/ent/enum"
	"gitlab.calendaria.team/services/finance/billing/ent/invoice"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
)

type InvoicesRepo interface {
	CreateInvoice(ctx context.Context, dto InvoiceDto) (*ent.Invoice, error)
	UpdateInvoice(ctx context.Context, invoiceData *ent.Invoice, dto InvoiceDto) (*ent.Invoice, error)
	DeleteInvoice(ctx context.Context, invoiceID int64) error
	GetInvoice(ctx context.Context, actorID, tenantID int64, appID string, invoiceID int64) (*ent.Invoice, error)
	CountInvoices(ctx context.Context, filter InvoiceFilter) (int32, error)
	ListInvoices(
		ctx context.Context, filter InvoiceFilter, paginate *utils_v1.PaginateRequest,
	) ([]*ent.Invoice, error)
	GetInvoicesToExpire(ctx context.Context, paidTill *time.Time) ([]*ent.Invoice, error)
	GetInvoicesToRevoke(ctx context.Context, paidTill *time.Time) ([]*ent.Invoice, error)
	GetInvoiceById(ctx context.Context, id int64) (*ent.Invoice, error)
}

type invoicesRepo struct {
	db *ent.Client
}

func NewInvoicesRepo(d *Data) InvoicesRepo {
	return &invoicesRepo{
		db: d.db,
	}
}

func (r *invoicesRepo) CreateInvoice(ctx context.Context, dto InvoiceDto) (*ent.Invoice, error) {
	query := r.db.Invoice.Create().
		SetUserID(dto.UserID).
		SetTenantID(dto.TenantID).
		SetAppID(dto.AppID).
		SetProductID(dto.ProductID).
		SetAmount(dto.Amount).
		SetPrice(dto.Price).
		SetStatus(dto.Status)

	if dto.SubscriptionID != 0 {
		query.SetSubscriptionsID(dto.SubscriptionID)
	}

	if dto.PaidAt != nil {
		query = query.SetPaidAt(*dto.PaidAt)
	}

	if dto.PaidTill != nil {
		query = query.SetPaidTill(*dto.PaidTill)
	}

	if dto.AppleStoreTransactionID != nil {
		query = query.SetAppleStoreTransactionID(*dto.AppleStoreTransactionID)
	}

	return query.Save(ctx)
}

func (r *invoicesRepo) UpdateInvoice(
	ctx context.Context, invoiceData *ent.Invoice, dto InvoiceDto,
) (*ent.Invoice, error) {
	query := r.db.Invoice.UpdateOne(invoiceData)

	if dto.Status.IsValid() {
		query.SetStatus(dto.Status)
	}

	if dto.PaidAt != nil {
		query = query.SetPaidAt(*dto.PaidAt)
	}

	if dto.IsPaidAtProcessed != nil {
		query = query.SetIsPaidAtProcessed(*dto.IsPaidAtProcessed)
	}

	if dto.IsPaidTillProcessed != nil {
		query = query.SetIsPaidTillProcessed(*dto.IsPaidTillProcessed)
	}

	if dto.IsRevoked != nil {
		query = query.SetIsRevoked(*dto.IsRevoked)
	}

	if dto.RevokedAt != nil {
		query = query.SetRevokedAt(*dto.RevokedAt)
	}

	if dto.IsRevokedProcessed != nil {
		query = query.SetIsRevokedProcessed(*dto.IsRevokedProcessed)
	}

	return query.Save(ctx)
}

func (r *invoicesRepo) DeleteInvoice(ctx context.Context, invoiceID int64) error {
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

func (r *invoicesRepo) GetInvoiceById(ctx context.Context, id int64) (*ent.Invoice, error) {
	return r.db.Invoice.Query().Where(invoice.ID(id)).Only(ctx)
}

func (r *invoicesRepo) CountInvoices(ctx context.Context, filter InvoiceFilter) (int32, error) {
	query := r.db.Invoice.Query()

	if filter.UserID != 0 {
		query.Where(invoice.UserID(filter.UserID))
	}

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

	//nolint:gosec // pagination limit cannot hold more than int32
	return int32(n), err
}

func (r *invoicesRepo) ListInvoices(
	ctx context.Context, filter InvoiceFilter, paginate *utils_v1.PaginateRequest,
) ([]*ent.Invoice, error) {
	query := r.db.Invoice.Query().Where(invoice.IDGT(paginate.GetFromId())).Limit(int(paginate.GetLimit()))

	if filter.UserID != 0 {
		query.Where(invoice.UserID(filter.UserID))
	}

	if filter.Status.IsValid() {
		query.Where(invoice.StatusEQ(filter.Status))
	}

	if filter.ProductID != 0 {
		query.Where(invoice.ProductIDEQ(filter.ProductID))
	}

	if filter.Paid {
		query.Where(invoice.PaidAtNotNil())
	}

	if filter.PaidProcesses != nil {
		query.Where(invoice.IsPaidAtProcessed(*filter.PaidProcesses))
	}

	if filter.PaidTillProc != nil {
		query.Where(invoice.IsPaidTillProcessed(*filter.PaidTillProc))
	}

	if filter.IsRevoked != nil {
		query.Where(invoice.IsRevoked(*filter.IsRevoked))
	}

	if filter.IsRevokedProc != nil {
		query.Where(invoice.IsRevokedProcessed(*filter.IsRevokedProc))
	}

	if filter.PaidTill != nil {
		query.Where(invoice.PaidTillGT(*filter.PaidTill))
	}

	if filter.SubscriptionID != 0 {
		query.Where(invoice.SubscriptionIDEQ(filter.SubscriptionID))
	}

	return query.All(ctx)
}

func (r *invoicesRepo) GetInvoicesToExpire(ctx context.Context, paidTill *time.Time) ([]*ent.Invoice, error) {
	return r.db.Invoice.Query().Where(
		invoice.StatusEQ(enum.Paid),
		invoice.IsPaidAtProcessed(true),
		invoice.IsRevoked(false),
		invoice.IsPaidTillProcessed(false),
		invoice.PaidTillLT(*paidTill),
	).Modify(func(s *sql.Selector) {
		invoicesT := sql.Table(invoice.Table).As("t2")

		s.LeftJoin(invoicesT).
			On(invoicesT.C(invoice.FieldSubscriptionID), s.C(invoice.FieldSubscriptionID)).
			OnP(sql.ColumnsLT(s.C(invoice.FieldPaidTill), invoicesT.C(invoice.FieldPaidTill)))
		s.Where(sql.IsNull(invoicesT.C(invoice.FieldPaidTill)))
	}).Limit(int(BackgroundProcessPageSize)).
		All(ctx)
}

func (r *invoicesRepo) GetInvoicesToRevoke(ctx context.Context, paidTill *time.Time) ([]*ent.Invoice, error) {
	return r.db.Invoice.Query().Where(
		invoice.StatusEQ(enum.Paid),
		invoice.IsPaidAtProcessed(true),
		invoice.IsRevoked(true),
		invoice.IsRevokedProcessed(false),
		invoice.IsPaidTillProcessed(false),
	).Modify(func(s *sql.Selector) {
		invoicesT := sql.Table(invoice.Table).As("t2")

		s.LeftJoin(invoicesT).
			On(invoicesT.C(invoice.FieldSubscriptionID), s.C(invoice.FieldSubscriptionID)).
			OnP(sql.ColumnsLT(s.C(invoice.FieldPaidTill), invoicesT.C(invoice.FieldPaidTill)))

		s.Where(sql.And(
			sql.IsNull(invoicesT.C(invoice.FieldPaidTill)),
			sql.NotNull(s.C(invoice.FieldPaidTill)),
			sql.GT(s.C(invoice.FieldPaidTill), paidTill),
			sql.ColumnsLT(s.C(invoice.FieldRevokedAt), s.C(invoice.FieldPaidTill)),
		))
	}).Limit(int(BackgroundProcessPageSize)).
		All(ctx)
}
