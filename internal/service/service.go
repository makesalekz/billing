package service

import (
	"github.com/google/wire"
	"gitlab.calendaria.team/services/finance/billing/internal/data"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
)

// ProviderSet is service providers.
//
//nolint:gochecknoglobals // global variable, used in wire
var ProviderSet = wire.NewSet(
	NewItemService,
	NewProductService,
	NewInvoiceService,
	NewSubscriptionService,
	NewAppleStoreService,
)

func FormPaginateRequest(req *utils_v1.PaginateRequest) *utils_v1.PaginateRequest {
	if req == nil {
		return &utils_v1.PaginateRequest{
			FromId: 0,
			Limit:  data.DefaultPageSize,
		}
	}

	paginateRequest := &utils_v1.PaginateRequest{
		FromId: req.GetFromId(),
		Limit:  req.GetLimit(),
	}

	if paginateRequest.GetFromId() == 0 {
		paginateRequest.FromId = 0
	}

	if paginateRequest.GetLimit() == 0 {
		paginateRequest.Limit = data.DefaultPageSize
	}

	return paginateRequest
}
