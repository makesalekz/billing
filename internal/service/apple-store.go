package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	v1 "gitlab.calendaria.team/services/finance/invoices/api/billing/v1"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
)

type AppleStoreService struct {
	log *log.Helper

	v1.UnimplementedAppleStoreServer
}

func NewAppleStoreService(logger log.Logger) *AppleStoreService {
	return &AppleStoreService{
		log: log.NewHelper(logger),
	}
}

func (s *AppleStoreService) ProcessServerNotification(
	ctx context.Context, req *v1.ProcessServerNotificationRequest,
) (*utils_v1.EmptyReply, error) {
	s.log.Info(req.GetSignedPayload())

	return &utils_v1.EmptyReply{}, nil
}
