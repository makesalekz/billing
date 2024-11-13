package data

import (
	"context"

	"gitlab.calendaria.team/services/finance/invoices/ent"
	"gitlab.calendaria.team/services/finance/invoices/ent/subscriptions"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
)

type SubscriptionsRepo interface {
	CreateSubscription(
		ctx context.Context, actorID, tenantID int64, appID string, dto *SubscriptionDto,
	) (*ent.Subscriptions, error)
	GetSubscription(
		ctx context.Context, actorID, tenantID int64, appID string, subscriptionID int64,
	) (*ent.Subscriptions, error)
	DeleteSubscription(ctx context.Context, actorID, subscriptionID int64) error
	CountSubscriptions(ctx context.Context, actorID int64) (int32, error)
	ListSubscriptions(
		ctx context.Context, actorID int64, paginate *utils_v1.PaginateRequest,
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
	ctx context.Context, actorID, tenantID int64, appID string, dto *SubscriptionDto,
) (*ent.Subscriptions, error) {
	return r.db.Subscriptions.Create().
		SetUserID(dto.UserID).
		SetTenantID(dto.TenantID).
		SetAppID(dto.AppID).
		Save(ctx)
}

func (r *subscriptionsRepo) GetSubscription(
	ctx context.Context, actorID, tenantID int64, appID string, subscriptionID int64,
) (*ent.Subscriptions, error) {
	return r.db.Subscriptions.Query().
		Where(
			subscriptions.ID(subscriptionID),
			subscriptions.UserID(actorID),
			subscriptions.TenantID(tenantID),
			subscriptions.AppID(appID),
		).
		Only(ctx)
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

	return int32(n), nil
}

func (r *subscriptionsRepo) ListSubscriptions(
	ctx context.Context, actorID int64, paginate *utils_v1.PaginateRequest,
) ([]*ent.Subscriptions, error) {
	return r.db.Subscriptions.Query().
		Where(
			subscriptions.UserID(actorID),
			subscriptions.IDGT(paginate.GetFromId()),
		).
		Limit(int(paginate.Limit)).
		All(ctx)
}
