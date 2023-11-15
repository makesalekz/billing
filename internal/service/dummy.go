package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	v1 "gitlab.calendaria.team/services/dummy/api/dummy/v1"
	"gitlab.calendaria.team/services/dummy/internal/biz"
)

type DummyService struct {
	v1.UnimplementedDummyServer

	log *log.Helper
	uc  *biz.DummyUsecase
}

func NewDummyService(logger log.Logger, uc *biz.DummyUsecase) *DummyService {
	return &DummyService{
		log: log.NewHelper(logger),
		uc:  uc,
	}
}

func (s *DummyService) DummyMethod(ctx context.Context, req *v1.DummyRequest) (*v1.DummyReply, error) {
	_, err := s.uc.DummyMethod(ctx)
	if err != nil {
		return nil, err
	}

	return &v1.DummyReply{}, nil
}
