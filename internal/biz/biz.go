package biz

import (
	"github.com/google/wire"

	"gitlab.calendaria.team/services/utils/v4/nats"
)

// ProviderSet is biz providers.
//
//nolint:gochecknoglobals // global variable, used in wire
var ProviderSet = wire.NewSet(
	nats.NewQueueManager,
	NewInvoicesManager,
	NewItemsUsecase,
	NewProductUseCase,
	NewInvoicesUseCase,
	NewSubscriptionUsecase,
	NewAppleStoreUsecase,
	NewPaymentUsecase,
	NewAppleStoreReliabilityUseCase,
)
