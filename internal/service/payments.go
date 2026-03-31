package service

import (
	"context"
	"strconv"

	v1 "gitlab.calendaria.team/services/finance/billing/api/billing/v1"
	"gitlab.calendaria.team/services/finance/billing/internal/biz"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
	"gitlab.calendaria.team/services/utils/v2/auth"
)

type PaymentsService struct {
	v1.UnimplementedPaymentsServer
	uc *biz.PaymentUseCase
}

func NewPaymentsService(
	uc *biz.PaymentUseCase,
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
		return nil, v1.ErrorEmptyActorId("actor id is empty")
	}

	tenantID := auth.GetTenantIdFromContext(ctx)
	if tenantID == 0 {
		return nil, v1.ErrorEmptyTenantId("tenant id is empty")
	}

	appID := auth.GetAppIdFromContext(ctx)
	if appID == "" {
		return nil, v1.ErrorEmptyAppId("app id is empty")
	}

	if req.GetProductId() == 0 {
		return nil, v1.ErrorInvalidRequest("product id is empty")
	}

	if req.GetCardCryptogramPacket() == "" {
		return nil, v1.ErrorInvalidRequest("card cryptogram is required")
	}

	if req.GetIpAddress() == "" {
		return nil, v1.ErrorInvalidRequest("ip address is required")
	}

	amount := req.GetAmount()
	if amount == 0 {
		amount = 1
	}

	resp, err := s.uc.CreatePayment(
		ctx, tenantID, actorID, req.GetProductId(), appID, amount,
		req.GetCardCryptogramPacket(), req.GetIpAddress(),
		req.GetName(), req.GetEmail(),
	)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *PaymentsService) Complete3DS(ctx context.Context, req *v1.Complete3DSRequest) (
	*v1.Complete3DSResponse, error,
) {
	if req.GetTransactionId() == 0 {
		return nil, v1.ErrorInvalidRequest("transaction id is required")
	}

	if req.GetPaRes() == "" {
		return nil, v1.ErrorInvalidRequest("paRes is required")
	}

	return s.uc.Complete3DS(ctx, req.GetTransactionId(), req.GetPaRes())
}

func (s *PaymentsService) CancelSubscription(
	ctx context.Context, req *v1.CancelSubscriptionRequest,
) (*utils_v1.EmptyReply, error) {
	err := s.uc.CancelSubscription(ctx, req.GetSubscriptionId())
	if err != nil {
		return nil, err
	}
	return &utils_v1.EmptyReply{}, nil
}

func (s *PaymentsService) GetPaymentStatus(ctx context.Context, req *v1.GetPaymentStatusRequest) (*v1.GetPaymentStatusResponse, error) {
	actorID := auth.GetActorIdFromContext(ctx)
	if actorID == 0 {
		return nil, v1.ErrorEmptyActorId("actor id is empty")
	}

	txID := strconv.FormatInt(req.GetTransactionId(), 10)
	return s.uc.GetPaymentStatus(ctx, txID, actorID)
}

func (s *PaymentsService) CheckWebhook(ctx context.Context, req *v1.WebhookPayload) (*v1.WebhookResponse, error) {
	code, message := s.uc.HandleCheckWebhook(ctx, req)
	return &v1.WebhookResponse{Code: int32(code), Message: message}, nil
}

func (s *PaymentsService) PaymentWebhook(ctx context.Context, req *v1.WebhookPayload) (*v1.WebhookResponse, error) {
	code, message := s.uc.HandlePaymentWebhook(ctx, req)
	return &v1.WebhookResponse{Code: int32(code), Message: message}, nil
}

func (s *PaymentsService) RecurrentWebhook(ctx context.Context, req *v1.WebhookPayload) (*v1.WebhookResponse, error) {
	code, message := s.uc.HandleRecurrentWebhook(ctx, req)
	return &v1.WebhookResponse{Code: int32(code), Message: message}, nil
}

// PaymentCallback handles legacy OVP callbacks (no-op).
func (s *PaymentsService) PaymentCallback(
	ctx context.Context, req *v1.PaymentCallbackRequest,
) (*utils_v1.EmptyReply, error) {
	s.uc.HandlePaymentCallback(ctx, req)
	return &utils_v1.EmptyReply{}, nil
}

// HTTP webhook handlers removed — webhooks are now parsed on BFF side
// and forwarded via gRPC as WebhookPayload
