package service

import (
	"context"

	v1 "gitlab.calendaria.team/services/finance/invoices/api/billing/v1"
	"gitlab.calendaria.team/services/finance/invoices/internal/biz"
	"gitlab.calendaria.team/services/finance/invoices/internal/data"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
)

type ItemService struct {
	v1.UnimplementedItemsServer

	uc *biz.ItemUseCase
}

func NewItemService(uc *biz.ItemUseCase) *ItemService {
	return &ItemService{uc: uc}
}

func (s *ItemService) CreateItem(ctx context.Context, req *v1.CreateItemRequest) (*v1.ItemReply, error) {
	itemDto := &data.ItemDto{
		Name:        req.GetItem().GetName(),
		Description: req.GetItem().GetDescription(),
	}

	item, err := s.uc.CreateItem(ctx, itemDto)
	if err != nil {
		return nil, err
	}

	return &v1.ItemReply{
		Item: biz.ReplyItem(item),
	}, nil
}

func (s *ItemService) UpdateItem(ctx context.Context, req *v1.UpdateItemRequest) (*v1.ItemReply, error) {
	itemDto := &data.ItemDto{
		Name:        req.GetItem().GetName(),
		Description: req.GetItem().GetDescription(),
	}

	item, err := s.uc.UpdateItem(ctx, req.GetId(), itemDto)
	if err != nil {
		return nil, err
	}

	return &v1.ItemReply{
		Item: biz.ReplyItem(item),
	}, nil
}

func (s *ItemService) DeleteItem(ctx context.Context, req *v1.ItemRequest) (*utils_v1.EmptyReply, error) {
	err := s.uc.DeleteItem(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &utils_v1.EmptyReply{}, nil
}

func (s *ItemService) GetItem(ctx context.Context, req *v1.ItemRequest) (*v1.ItemReply, error) {
	item, err := s.uc.GetItem(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &v1.ItemReply{
		Item: biz.ReplyItem(item),
	}, nil
}

func (s *ItemService) ListItems(ctx context.Context, req *v1.ListItemsRequest) (*v1.ListItemsReply, error) {
	itemList, err := s.uc.ListItems(ctx, req.GetPagination())
	if err != nil {
		return nil, err
	}

	return &v1.ListItemsReply{
		Items:      biz.ReplyItems(itemList.Items),
		Pagination: itemList.PaginateReply,
	}, nil
}
