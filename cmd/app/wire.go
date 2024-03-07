//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"gitlab.calendaria.team/services/dummy/internal/biz"
	"gitlab.calendaria.team/services/dummy/internal/conf"
	"gitlab.calendaria.team/services/dummy/internal/data"
	"gitlab.calendaria.team/services/dummy/internal/server"
	"gitlab.calendaria.team/services/dummy/internal/service"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// wireApp init kratos application.
func wireApp(*conf.Bootstrap, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
