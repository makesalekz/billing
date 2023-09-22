package service

import (
	"context"

	v1 "dummy/api/dummy/v1"
	"dummy/internal/biz"
	"dummy/internal/data"

	"github.com/go-kratos/kratos/v2/log"
)

type DummyService struct {
	v1.UnimplementedDummyServer

	log *log.Helper
	jwt *data.JwtProcessor
	uc  *biz.DummyUsecase
}

func NewUploadService(logger log.Logger, jwt *data.JwtProcessor, uc *biz.DummyUsecase) *DummyService {
	return &DummyService{
		log: log.NewHelper(logger),
		jwt: jwt,
		uc:  uc,
	}
}

func (s *DummyService) DummyMethod(ctx context.Context, req *v1.DummyRequest) (*v1.DummyReply, error) {
	userId, ok := s.jwt.GetUserIdFromContext(ctx)
	if !ok {
		return nil, v1.ErrorUnauthorized("Unauthorized")
	}

	_, err := s.uc.DummyMethod(ctx, userId)
	if err != nil {
		return nil, err
	}

	return &v1.DummyReply{}, nil
}
