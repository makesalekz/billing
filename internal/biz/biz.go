package biz

import (
	"github.com/google/wire"

	"gitlab.calendaria.team/services/utils/v2/nats"
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
	NewAppleStoreUsecase,
	NewPaymentUsecase,
)

const (
	DefaultPaymentCurrency          = "KZT"
	DefaultPaymentLifeTime          = 60 * 60 * 1            // 1 hour
	DefaultRecurrentProfileLifeTime = 60 * 60 * 24 * 365 * 4 // 4 years
	DefaultPaymentLang              = "ru"
	DefaultPriceForCardLink         = 10
	PmsAppID                        = "pms"
)
