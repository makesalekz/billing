package service

import (
	"github.com/google/wire"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
)

const DefaultPageSize = 100

// ProviderSet is service providers.
var ProviderSet = wire.NewSet(
	NewItemService,
)

func FormPaginateRequest(req *utils_v1.PaginateRequest) *utils_v1.PaginateRequest {
	if req == nil {
		return &utils_v1.PaginateRequest{
			FromId: 0,
			Limit:  DefaultPageSize,
		}
	}

	paginateRequest := &utils_v1.PaginateRequest{
		FromId: req.GetFromId(),
		Limit:  req.GetLimit(),
	}

	if paginateRequest.FromId == 0 {
		paginateRequest.FromId = 0
	}

	if paginateRequest.Limit == 0 {
		paginateRequest.Limit = DefaultPageSize
	}

	return paginateRequest
}
