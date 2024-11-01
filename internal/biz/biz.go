package biz

import (
	"github.com/google/wire"
	"gitlab.calendaria.team/services/utils/v1/nats"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
	nats.NewQueueManager,
	NewItemsUsecase,
)
