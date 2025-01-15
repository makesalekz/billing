package data

import (
	"context"
	"time"

	"gitlab.calendaria.team/services/finance/billing/ent"
	"gitlab.calendaria.team/services/finance/billing/ent/enum"
	"gitlab.calendaria.team/services/finance/billing/ent/invoice"
	"gitlab.calendaria.team/services/finance/billing/ent/subscriptions"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
)

type SubscriptionsRepo interface {
	CreateSubscription(
		ctx context.Context, actorID, tenantID int64, appID string, dto SubscriptionDto,
	) (*ent.Subscriptions, error)
	GetSubscription(
		ctx context.Context, actorID, tenantID int64, appID string, subscriptionID int64, withInvoices bool,
	) (*ent.Subscriptions, error)
	GetSubscriptionByOriginalAppleTransactionID(
		ctx context.Context, originalTransactionID string, withInvoices bool,
	) (*ent.Subscriptions, error)
	RevokeActiveSubscription(
		ctx context.Context, subscriptionID int64, revokedAt time.Time,
	) error
	DeleteSubscription(ctx context.Context, actorID, subscriptionID int64) error
	CountSubscriptions(ctx context.Context, actorID int64) (int32, error)
	ListSubscriptions(
		ctx context.Context, actorID int64, withInvoices bool, paginate *utils_v1.PaginateRequest,
	) ([]*ent.Subscriptions, error)
}

type subscriptionsRepo struct {
	db *ent.Client
}

func NewSubscriptionsRepo(d *Data) SubscriptionsRepo {
	return &subscriptionsRepo{
		db: d.db,
	}
}

func (r *subscriptionsRepo) CreateSubscription(
	ctx context.Context, actorID, tenantID int64, appID string, dto SubscriptionDto,
) (*ent.Subscriptions, error) {
	return r.db.Subscriptions.Create().
		SetUserID(actorID).
		SetTenantID(tenantID).
		SetAppID(appID).
		SetProductID(dto.ProductID).
		Save(ctx)
}

func (r *subscriptionsRepo) GetSubscription(
	ctx context.Context, actorID, tenantID int64, appID string, subscriptionID int64, withInvoices bool,
) (*ent.Subscriptions, error) {
	query := r.db.Subscriptions.Query().
		Where(
			subscriptions.ID(subscriptionID),
			subscriptions.UserID(actorID),
			subscriptions.TenantID(tenantID),
			subscriptions.AppID(appID),
		)

	if withInvoices {
		query = query.WithInvoices()
	}

	return query.Only(ctx)
}

func (r *subscriptionsRepo) GetSubscriptionByOriginalAppleTransactionID(
	ctx context.Context, originalTransactionID string, withInvoices bool,
) (*ent.Subscriptions, error) {
	query := r.db.Subscriptions.Query().
		Where(
			subscriptions.HasInvoicesWith(
				invoice.ExternalTransactionID(originalTransactionID),
				invoice.PaymentProviderEQ(enum.AppStore),
			),
		)

	if withInvoices {
		query = query.WithInvoices()
	}

	return query.Only(ctx)
}

func (r *subscriptionsRepo) RevokeActiveSubscription(
	ctx context.Context, subscriptionID int64, revokedAt time.Time,
) error {
	return r.db.Invoice.Update().Where(
		invoice.SubscriptionID(subscriptionID),
		invoice.PaidTillGT(revokedAt),
	).SetIsRevoked(true).
		SetRevokedAt(revokedAt).
		Exec(ctx)
}

func (r *subscriptionsRepo) DeleteSubscription(ctx context.Context, actorID, subscriptionID int64) error {
	return r.db.Subscriptions.
		DeleteOneID(subscriptionID).
		Where(subscriptions.UserID(actorID)).
		Exec(ctx)
}

func (r *subscriptionsRepo) CountSubscriptions(ctx context.Context, actorID int64) (int32, error) {
	n, err := r.db.Subscriptions.Query().
		Where(subscriptions.UserID(actorID)).
		Count(ctx)
	if err != nil {
		return 0, err
	}

	//nolint:gosec // pagination limit cannot hold more than int32
	return int32(n), nil
}

func (r *subscriptionsRepo) ListSubscriptions(
	ctx context.Context, actorID int64, withInvoices bool, paginate *utils_v1.PaginateRequest,
) ([]*ent.Subscriptions, error) {
	return r.db.Subscriptions.Query().
		Where(
			subscriptions.UserID(actorID),
			subscriptions.IDGT(paginate.GetFromId()),
		).
		WithInvoices().
		Limit(int(paginate.GetLimit())).
		All(ctx)
}
