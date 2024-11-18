package biz

import (
	"context"

	v1 "gitlab.calendaria.team/services/finance/billing/api/billing/v1"
	"gitlab.calendaria.team/services/finance/billing/ent"
	"gitlab.calendaria.team/services/finance/billing/internal/data"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
)

type ItemList struct {
	Items         []*ent.Item
	PaginateReply *utils_v1.PaginateReply
}

type ItemUseCase struct {
	itemsRepo data.ItemsRepo
}

func NewItemsUsecase(itemsRepo data.ItemsRepo) *ItemUseCase {
	return &ItemUseCase{itemsRepo: itemsRepo}
}

func (uc *ItemUseCase) CreateItem(ctx context.Context, itemDto data.ItemDto) (*ent.Item, error) {
	item, err := uc.itemsRepo.CreateItem(ctx, itemDto)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("failed to create item: %s", err.Error())
	}

	return item, nil
}

func (uc *ItemUseCase) UpdateItem(ctx context.Context, itemID int64, itemDto data.ItemDto) (*ent.Item, error) {
	item, err := uc.itemsRepo.UpdateItem(ctx, itemID, itemDto)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("failed to update item: %s", err.Error())
	}

	return item, nil
}

func (uc *ItemUseCase) DeleteItem(ctx context.Context, itemID int64) error {
	err := uc.itemsRepo.DeleteItem(ctx, itemID)
	if err != nil {
		return v1.ErrorDatabaseQuery("failed to delete item: %s", err.Error())
	}

	return nil
}

func (uc *ItemUseCase) GetItem(ctx context.Context, itemID int64) (*ent.Item, error) {
	item, err := uc.itemsRepo.GetItem(ctx, itemID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, v1.ErrorNotFound("failed to get item: %s", err.Error())
		}

		return nil, v1.ErrorDatabaseQuery("failed to get item: %s", err.Error())
	}

	return item, nil
}

func (uc *ItemUseCase) ListItems(ctx context.Context, paginate *utils_v1.PaginateRequest) (*ItemList, error) {
	count, err := uc.itemsRepo.CountItems(ctx)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("failed to count items: %s", err.Error())
	}

	items, err := uc.itemsRepo.ListItems(ctx, paginate)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("failed to list items: %s", err.Error())
	}

	paginateReply := &utils_v1.PaginateReply{
		Total: &count,
	}

	if len(items) > 0 {
		paginateReply.FromId = &items[len(items)-1].ID
	}

	return &ItemList{
		Items:         items,
		PaginateReply: paginateReply,
	}, nil
}

func ReplyItem(item *ent.Item) *v1.Item {
	return &v1.Item{
		Id:          item.ID,
		Name:        item.Name,
		Description: item.Description,
	}
}

func ReplyItems(items []*ent.Item) []*v1.Item {
	replyItems := make([]*v1.Item, len(items))
	for i, item := range items {
		replyItems[i] = ReplyItem(item)
	}
	return replyItems
}
