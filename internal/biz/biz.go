package biz

import (
	"github.com/google/wire"
	"gitlab.calendaria.team/services/utils/v1/nats"
)

// ProviderSet is biz providers.
//
//nolint:gochecknoglobals // global variable, used in wire
var ProviderSet = wire.NewSet(
	nats.NewQueueManager,
	NewItemsUsecase,
	NewProductUseCase,
	NewInvoicesUseCase,
	NewSubscriptionUsecase,
)

const (
	DefaultPageSize int32 = 100
)
