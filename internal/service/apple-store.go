package service

import (
	"context"
	"encoding/json"

	v1 "gitlab.calendaria.team/services/finance/billing/api/billing/v1"
	"gitlab.calendaria.team/services/finance/billing/internal/biz"
	"gitlab.calendaria.team/services/finance/billing/internal/data"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
	"gitlab.calendaria.team/services/utils/v2/auth"

	"github.com/golang-jwt/jwt/v5"
)

type AppleStoreService struct {
	v1.UnimplementedAppleStoreServer

	uc        *biz.AppleStoreUsecase
	JWTParser data.JWTParser
}

func NewAppleStoreService(uc *biz.AppleStoreUsecase) *AppleStoreService {
	return &AppleStoreService{
		uc:        uc,
		JWTParser: data.NewDefaultJWTParser(),
	}
}

func (s *AppleStoreService) ProcessServerNotification(
	ctx context.Context, req *v1.ProcessServerNotificationRequest,
) (*utils_v1.EmptyReply, error) {
	decodedPayload, err := s.JWTParser.ParseAppleSignedBody(req.GetSignedPayload())
	if err != nil {
		return nil, v1.ErrorInvalidRequest("failed to parse signed payload: %s", err.Error())
	}

	if !decodedPayload.Valid {
		return nil, v1.ErrorForbidden("invalid signed payload")
	}

	var payload data.Payload

	mapClaims, ok := decodedPayload.Claims.(jwt.MapClaims)
	if !ok {
		return nil, v1.ErrorInvalidRequest("failed to parse signed payload: claims is not a map")
	}

	jsonBody, err := json.Marshal(mapClaims)
	if err != nil {
		return nil, v1.ErrorInvalidRequest("failed to parse signed payload: %s", err.Error())
	}

	err = json.Unmarshal(jsonBody, &payload)
	if err != nil {
		return nil, v1.ErrorInvalidRequest("failed to parse signed payload: %s", err.Error())
	}

	err = s.uc.ProcessPayload(ctx, payload)
	if err != nil {
		return nil, err
	}

	return &utils_v1.EmptyReply{}, nil
}

func (s *AppleStoreService) ClientNotification(
	ctx context.Context, req *v1.ProcessServerNotificationRequest,
) (*v1.ClientNotificationReply, error) {
	actorID := auth.GetActorIdFromContext(ctx)
	if actorID == 0 {
		return nil, v1.ErrorUnauthorized("actor ID is required")
	}

	tenantID := auth.GetTenantIdFromContext(ctx)
	if tenantID == 0 {
		return nil, v1.ErrorUnauthorized("tenant ID is required")
	}

	decodedPayload, err := s.JWTParser.ParseAppleSignedBody(req.GetSignedPayload())
	if err != nil {
		return nil, v1.ErrorInvalidRequest("failed to parse signed payload: %s", err.Error())
	}

	if !decodedPayload.Valid {
		return nil, v1.ErrorForbidden("invalid signed payload")
	}

	var transaction data.JWSTransaction
	mapClaims, ok := decodedPayload.Claims.(jwt.MapClaims)
	if !ok {
		return nil, v1.ErrorInvalidRequest("failed to parse signed payload: claims is not a map")
	}

	jsonBody, err := json.Marshal(mapClaims)
	if err != nil {
		return nil, v1.ErrorInvalidRequest("failed to parse signed payload: %s", err.Error())
	}

	err = json.Unmarshal(jsonBody, &transaction)
	if err != nil {
		return nil, v1.ErrorInvalidRequest("failed to parse signed payload: %s", err.Error())
	}

	if transaction.OriginalTransactionID == "" {
		return nil, v1.ErrorInvalidRequest("original transaction ID is required")
	}

	invoice, subscription, err := s.uc.ClientNotification(ctx, actorID, tenantID, transaction)
	if err != nil {
		return nil, err
	}

	return &v1.ClientNotificationReply{
		Invoice:      biz.ReplyInvoice(invoice),
		Subscription: biz.ReplySubscription(subscription),
	}, nil
}
