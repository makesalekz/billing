package data

import (
	"context"

	"gitlab.calendaria.team/services/finance/invoices/ent"
	"gitlab.calendaria.team/services/finance/invoices/ent/item"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
)

type ItemsRepo interface {
	CreateItem(ctx context.Context, item *ItemDto) (*ent.Item, error)
	UpdateItem(ctx context.Context, itemID int64, item *ItemDto) (*ent.Item, error)
	DeleteItem(ctx context.Context, itemID int64) error
	GetItem(ctx context.Context, itemID int64) (*ent.Item, error)
	GetItems(ctx context.Context, itemIDs []int64) ([]*ent.Item, error)
	CountItems(ctx context.Context) (int64, error)
	ListItems(ctx context.Context, paginate *utils_v1.PaginateRequest) ([]*ent.Item, error)
}

type itemsRepo struct {
	db *ent.Client
}

func NewItemsRepo(d *Data) ItemsRepo {
	return &itemsRepo{
		db: d.db,
	}
}

func (r *itemsRepo) CreateItem(ctx context.Context, item *ItemDto) (*ent.Item, error) {
	return r.db.Item.Create().
		SetName(item.Name).
		SetDescription(item.Description).
		SetNillableTopicName(item.TopicName).
		Save(ctx)
}

func (r *itemsRepo) UpdateItem(ctx context.Context, itemID int64, item *ItemDto) (*ent.Item, error) {
	return r.db.Item.UpdateOneID(itemID).
		SetName(item.Name).
		SetDescription(item.Description).
		SetNillableTopicName(item.TopicName).
		Save(ctx)
}

func (r *itemsRepo) DeleteItem(ctx context.Context, itemID int64) error {
	return r.db.Item.DeleteOneID(itemID).
		Exec(ctx)
}

func (r *itemsRepo) GetItem(ctx context.Context, itemID int64) (*ent.Item, error) {
	return r.db.Item.Query().
		Where(item.ID(itemID)).
		Only(ctx)
}

func (r *itemsRepo) GetItems(ctx context.Context, itemIDs []int64) ([]*ent.Item, error) {
	return r.db.Item.Query().
		Where(item.IDIn(itemIDs...)).
		All(ctx)
}

func (r *itemsRepo) CountItems(ctx context.Context) (int64, error) {
	n, err := r.db.Item.Query().Count(ctx)
	if err != nil {
		return 0, err
	}

	return int64(n), nil
}

func (r *itemsRepo) ListItems(ctx context.Context, paginate *utils_v1.PaginateRequest) ([]*ent.Item, error) {
	return r.db.Item.Query().
		Where(item.IDGT(paginate.GetFromId())).
		Limit(int(paginate.GetLimit())).
		All(ctx)
}
