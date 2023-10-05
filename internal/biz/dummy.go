package biz

import (
	"context"
	_ "embed"

	dummy_v1 "dummy/api/dummy/v1"
	"dummy/ent"
	"dummy/internal/conf"
	"dummy/internal/data"

	consul "github.com/go-kratos/consul/registry"
	"github.com/go-kratos/kratos/v2/log"
)

// DummyUsecase is a Greeter usecase.
type DummyUsecase struct {
	conf      *conf.Bootstrap
	log       *log.Helper
	discovery *consul.Registry
	jwt       *data.JwtProcessor
	repo      data.DummyRepo
}

// NewGreeterUsecase new a Greeter usecase.
func NewDummyUsecase(logger log.Logger, c *data.Config, jwt *data.JwtProcessor, repo data.DummyRepo) (*DummyUsecase, error) {
	return &DummyUsecase{
		conf:      c.Bootstrap,
		log:       log.NewHelper(logger),
		discovery: c.GetRegistry(),
		jwt:       jwt,
		repo:      repo,
	}, nil
}

func (uc *DummyUsecase) DummyMethod(ctx context.Context) (*ent.Dummy, error) {
	userId, ok := uc.jwt.GetUserIdFromContext(ctx)
	if !ok {
		return nil, dummy_v1.ErrorUnauthorized("Unauthorized")
	}

	return uc.repo.CreateDummy(ctx, userId)
}
