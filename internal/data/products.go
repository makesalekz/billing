package data

import (
	"context"

	"gitlab.calendaria.team/services/finance/invoices/ent"
	"gitlab.calendaria.team/services/finance/invoices/ent/product"
)

type ProductRepo interface {
	CreateProduct(ctx context.Context, product *ProductDto) (*ent.Product, error)
	UpdateProduct(ctx context.Context, productID int64, product *ProductDto) (*ent.Product, error)
	DeleteProduct(ctx context.Context, productID int64) error
	GetProduct(ctx context.Context, productID int64) (*ent.Product, error)
}

type productsRepo struct {
	db *ent.Client
}

func NewProductsRepo(d *Data) ProductRepo {
	return &productsRepo{
		db: d.db,
	}
}

func (r *productsRepo) CreateProduct(ctx context.Context, productDto *ProductDto) (*ent.Product, error) {
	query := r.db.Product.Create().
		SetName(productDto.Name).
		SetDescription(productDto.Description).
		SetCurrency(productDto.Currency).
		SetPrice(productDto.Price).
		SetIsActive(productDto.IsActive).
		SetIsLimited(productDto.IsLimited).
		SetLeft(productDto.Left).
		SetIsUnique(productDto.IsUnique).
		SetUniqueLimit(productDto.UniqueLimit)

	if productDto.LimitedTill != nil {
		query.SetLimitedTill(*productDto.LimitedTill)
	}

	return query.Save(ctx)
}

func (r *productsRepo) UpdateProduct(ctx context.Context, productID int64, productDto *ProductDto) (
	*ent.Product, error,
) {
	query := r.db.Product.UpdateOneID(productID).
		SetName(productDto.Name).
		SetDescription(productDto.Description).
		SetCurrency(productDto.Currency).
		SetPrice(productDto.Price).
		SetIsActive(productDto.IsActive).
		SetIsLimited(productDto.IsLimited).
		SetLeft(productDto.Left).
		SetIsUnique(productDto.IsUnique).
		SetUniqueLimit(productDto.UniqueLimit)

	if productDto.LimitedTill != nil {
		query.SetLimitedTill(*productDto.LimitedTill)
	}

	return query.Save(ctx)
}

func (r *productsRepo) DeleteProduct(ctx context.Context, productID int64) error {
	return r.db.Product.DeleteOneID(productID).
		Exec(ctx)
}

func (r *productsRepo) GetProduct(ctx context.Context, productID int64) (*ent.Product, error) {
	return r.db.Product.Query().
		Where(product.ID(productID)).
		Only(ctx)
}
