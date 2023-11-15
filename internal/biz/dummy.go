package biz

import (
	"context"
	_ "embed"

	consul "github.com/go-kratos/consul/registry"
	"github.com/go-kratos/kratos/v2/log"
	dummy_v1 "gitlab.calendaria.team/services/dummy/api/dummy/v1"
	"gitlab.calendaria.team/services/dummy/ent"
	"gitlab.calendaria.team/services/dummy/internal/conf"
	"gitlab.calendaria.team/services/dummy/internal/data"
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
