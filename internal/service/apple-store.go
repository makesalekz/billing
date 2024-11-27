package service

import (
	"context"
	"encoding/json"

	"github.com/golang-jwt/jwt/v5"
	v1 "gitlab.calendaria.team/services/finance/billing/api/billing/v1"
	"gitlab.calendaria.team/services/finance/billing/internal/biz"
	"gitlab.calendaria.team/services/finance/billing/internal/data"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
)

type AppleStoreService struct {
	v1.UnimplementedAppleStoreServer

	uc *biz.AppleStoreUsecase
}

func NewAppleStoreService(uc *biz.AppleStoreUsecase) *AppleStoreService {
	return &AppleStoreService{
		uc: uc,
	}
}

func (s *AppleStoreService) ProcessServerNotification(
	ctx context.Context, req *v1.ProcessServerNotificationRequest,
) (*utils_v1.EmptyReply, error) {
	parse, err := data.ParseAppleSignedBody(req.GetSignedPayload())
	if err != nil {
		return nil, v1.ErrorInvalidRequest("failed to parse signed payload: %s", err.Error())
	}

	if !parse.Valid {
		return nil, v1.ErrorForbidden("invalid signed payload")
	}

	var payload data.Payload

	mapClaims, ok := parse.Claims.(jwt.MapClaims)
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
