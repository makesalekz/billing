package biz

import (
	"github.com/google/wire"

	"github.com/makesalekz/utils/v2/nats"
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
)
