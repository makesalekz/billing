package service

import (
	"context"

	v1 "gitlab.calendaria.team/services/finance/billing/api/billing/v1"
	"gitlab.calendaria.team/services/finance/billing/internal/biz"
	"gitlab.calendaria.team/services/utils/v2/auth"
)

type SubscriptionService struct {
	v1.UnimplementedSubscriptionsServer

	uc *biz.SubscriptionsUseCase
}

func NewSubscriptionService(uc *biz.SubscriptionsUseCase) *SubscriptionService {
	return &SubscriptionService{uc: uc}
}

func (s *SubscriptionService) GetSubscription(ctx context.Context, req *v1.GetSubscriptionRequest) (
	*v1.SubscriptionReply, error,
) {
	actorID := auth.GetActorIdFromContext(ctx)
	if actorID == 0 {
		return nil, v1.ErrorEmptyActorId("empty actor id")
	}

	tenantID := auth.GetTenantIdFromContext(ctx)
	if tenantID == 0 {
		return nil, v1.ErrorEmptyTenantId("empty tenant id")
	}

	appID := auth.GetAppIdFromContext(ctx)
	if appID == "" {
		return nil, v1.ErrorEmptyAppId("empty app id")
	}

	subscription, err := s.uc.GetSubscription(ctx, actorID, tenantID, appID, req.GetSubscriptionId(),
		req.GetWithInvoices())
	if err != nil {
		return nil, err
	}

	return &v1.SubscriptionReply{
		Subscription: biz.ReplySubscription(subscription),
	}, nil
}

func (s *SubscriptionService) ListSubscriptions(
	ctx context.Context, req *v1.ListSubscriptionsRequest,
) (*v1.ListSubscriptionsReply, error) {
	actorID := auth.GetActorIdFromContext(ctx)
	if actorID == 0 {
		return nil, v1.ErrorEmptyActorId("empty actor id")
	}

	tenantID := auth.GetTenantIdFromContext(ctx)
	if tenantID == 0 {
		return nil, v1.ErrorEmptyTenantId("empty tenant id")
	}

	appID := auth.GetAppIdFromContext(ctx)
	if appID == "" {
		return nil, v1.ErrorEmptyAppId("empty app id")
	}

	pagination := FormPaginateRequest(req.GetPagination())

	listSubscription, err := s.uc.ListSubscriptions(ctx, actorID, req.GetWithInvoices(), pagination)
	if err != nil {
		return nil, err
	}

	return &v1.ListSubscriptionsReply{
		Subscriptions: biz.ReplySubscriptions(listSubscription.Subscriptions),
		Pagination:    listSubscription.PaginateReply,
	}, nil
}

func (s *SubscriptionService) GetSubscriptionStatus(ctx context.Context, req *v1.GetSubscriptionStatusRequest) (
	*v1.SubscriptionStatusReply, error,
) {
	actorID := auth.GetActorIdFromContext(ctx)
	if actorID == 0 {
		return nil, v1.ErrorEmptyActorId("empty actor id")
	}
	tenantID := auth.GetTenantIdFromContext(ctx)
	if tenantID == 0 {
		return nil, v1.ErrorEmptyTenantId("empty tenant id")
	}
	appID := auth.GetAppIdFromContext(ctx)
	if appID == "" {
		return nil, v1.ErrorEmptyAppId("empty app id")
	}

	return s.uc.GetSubscriptionStatus(ctx, actorID, tenantID, appID)
}
