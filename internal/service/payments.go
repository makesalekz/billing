package service

import (
	"context"

	v1 "gitlab.calendaria.team/services/finance/billing/api/billing/v1"
	"gitlab.calendaria.team/services/finance/billing/internal/biz"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
	"gitlab.calendaria.team/services/utils/v2/auth"
)

type PaymentsService struct {
	v1.UnimplementedPaymentsServer
	uc *biz.PaymentUsecase
}

func NewPaymentsService(
	uc *biz.PaymentUsecase,
) *PaymentsService {
	return &PaymentsService{
		uc: uc,
	}
}

func (s *PaymentsService) CreatePayment(ctx context.Context, req *v1.CreatePaymentRequest) (
	*v1.CreatePaymentResponse, error,
) {
	actorID := auth.GetActorIdFromContext(ctx)
	if actorID == 0 {
		// return nil, v1.ErrorEmptyActorId("actor id is empty")
		actorID = 3
	}

	tenantID := auth.GetTenantIdFromContext(ctx)
	if tenantID == 0 {
		// return nil, v1.ErrorEmptyTenantId("tenant id is empty")
		tenantID = 3
	}

	appID := auth.GetAppIdFromContext(ctx)
	if appID == "" {
		// return nil, v1.ErrorEmptyAppId("app id is empty")
		appID = "pms"
	}

	if req.ProductId == 0 {
		return nil, v1.ErrorInvalidRequest("product id is empty")
	}

	invoiceID, paymentPageUrl, err := s.uc.CreatePayment(ctx, tenantID, actorID, req.ProductId, appID)
	if err != nil {
		return nil, err
	}

	return &v1.CreatePaymentResponse{
		InvoiceId:  invoiceID,
		PaymentUrl: paymentPageUrl,
	}, nil
}

func (s *PaymentsService) CancelSubscription(
	ctx context.Context, req *v1.CancelSubscriptionRequest,
) (*utils_v1.EmptyReply, error) {
	err := s.uc.CancelSubscription(ctx, req.SubscriptionId)
	if err != nil {
		return nil, err
	}
	return &utils_v1.EmptyReply{}, nil
}

func (s *PaymentsService) PaymentCallback(
	ctx context.Context, req *v1.PaymentCallbackRequest,
) (*utils_v1.EmptyReply, error) {
	s.uc.HandlePaymentCallback(ctx, req)
	return &utils_v1.EmptyReply{}, nil
}
