package biz

import (
	"context"

	v1 "gitlab.calendaria.team/services/finance/invoices/api/invoices/v1"
	"gitlab.calendaria.team/services/finance/invoices/ent"
	"gitlab.calendaria.team/services/finance/invoices/internal/data"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
)

type SubscriptionList struct {
	Subscriptions []*ent.Subscriptions
	PaginateReply *utils_v1.PaginateReply
}

type SubscriptionsUseCase struct {
	subscriptionRepo data.SubscriptionsRepo
}

func NewSubscriptionRepo(
	subscriptionRepo data.SubscriptionsRepo,
) *SubscriptionsUseCase {
	return &SubscriptionsUseCase{
		subscriptionRepo: subscriptionRepo,
	}
}

func (uc *SubscriptionsUseCase) CreateSubscription(
	ctx context.Context, actorID, tenantID int64, appID string, subscriptionDto *data.SubscriptionDto,
) (*ent.Subscriptions, error) {
	subscription, err := uc.subscriptionRepo.CreateSubscription(ctx, actorID, tenantID, appID, subscriptionDto)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("failed to create subscription, err %s", err.Error())
	}

	return subscription, nil
}

func (uc *SubscriptionsUseCase) GetSubscription(
	ctx context.Context, actorID, tenantID int64, appID string, subscriptionID int64,
) (*ent.Subscriptions, error) {
	subscription, err := uc.subscriptionRepo.GetSubscription(ctx, actorID, tenantID, appID, subscriptionID)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("failed to get subscription, err %s", err.Error())
	}

	return subscription, nil
}

func (uc *SubscriptionsUseCase) DeleteSubscription(ctx context.Context, actorID, subscriptionID int64) error {
	err := uc.subscriptionRepo.DeleteSubscription(ctx, actorID, subscriptionID)
	if err != nil {
		return v1.ErrorDatabaseQuery("failed to delete subscription, err %s", err.Error())
	}

	return nil
}

func (uc *SubscriptionsUseCase) ListSubscriptions(
	ctx context.Context, actorID int64, paginate *utils_v1.PaginateRequest,
) (*SubscriptionList, error) {
	subscriptions, err := uc.subscriptionRepo.ListSubscriptions(ctx, actorID, paginate)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("failed to list subscriptions, err %s", err.Error())
	}

	total, err := uc.subscriptionRepo.CountSubscriptions(ctx, actorID)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("failed to list subscriptions ,err %s", err.Error())
	}

	paginateReply := &utils_v1.PaginateReply{
		Total: &total,
	}

	if len(subscriptions) > 0 {
		paginateReply.FromId = &subscriptions[len(subscriptions)-1].ID
	}

	return &SubscriptionList{
		Subscriptions: subscriptions,
		PaginateReply: paginateReply,
	}, nil
}
