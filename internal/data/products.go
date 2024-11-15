package data

import (
	"context"

	v1 "gitlab.calendaria.team/services/finance/invoices/api/billing/v1"
	"gitlab.calendaria.team/services/finance/invoices/ent"
	"gitlab.calendaria.team/services/finance/invoices/ent/bundle"
	"gitlab.calendaria.team/services/finance/invoices/ent/product"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
)

type ProductRepo interface {
	CreateProduct(ctx context.Context, product *ProductDto) (*ent.Product, error)
	UpdateProduct(ctx context.Context, productEnt *ent.Product, product *ProductDto) (*ent.Product, error)
	DeleteProduct(ctx context.Context, productID int64) error
	GetProduct(ctx context.Context, productID int64) (*ent.Product, error)
	ListProducts(ctx context.Context, appID string, paginate *utils_v1.PaginateRequest) ([]*ent.Product, error)
	CountProducts(ctx context.Context, appID string) (int32, error)
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
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("transaction initialize failed")
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	query := tx.Product.Create().
		SetAppID(productDto.AppID).
		SetName(productDto.Name).
		SetDescription(productDto.Description).
		SetCurrency(productDto.Currency).
		SetPrice(productDto.Price).
		SetIsActive(productDto.IsActive).
		SetIsLimited(productDto.IsLimited).
		SetLeft(productDto.Left).
		SetIsUnique(productDto.IsUnique).
		SetUniqueLimit(productDto.UniqueLimit).
		SetIsExpiring(productDto.IsExpiring)

	if productDto.LimitedTill != nil {
		query.SetLimitedTill(*productDto.LimitedTill)
	}

	if productDto.ExpiringTime != nil {
		query.SetExpiringTime(*productDto.ExpiringTime)
	}

	product, err := query.Save(ctx)
	if err != nil {
		return nil, err
	}

	if len(productDto.Bundles) > 0 {
		var bundles []*ent.Bundle
		var bundleCreate []*ent.BundleCreate

		for _, bund := range productDto.Bundles {
			bundleCreate = append(bundleCreate, tx.Bundle.Create().
				SetAmount(bund.Amount).
				SetItemID(bund.ItemID).
				SetProductID(product.ID))
		}

		bundles, err = tx.Bundle.CreateBulk(bundleCreate...).Save(ctx)
		if err != nil {
			return nil, err
		}

		product.Edges.Bundles = bundles
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return product, nil
}

func (r *productsRepo) UpdateProduct(ctx context.Context, productEnt *ent.Product, productDto *ProductDto) (
	*ent.Product, error,
) {
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("transaction initialize failed")
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	query := tx.Product.UpdateOne(productEnt).
		SetName(productDto.Name).
		SetDescription(productDto.Description).
		SetCurrency(productDto.Currency).
		SetPrice(productDto.Price).
		SetIsActive(productDto.IsActive).
		SetIsLimited(productDto.IsLimited).
		SetLeft(productDto.Left).
		SetIsUnique(productDto.IsUnique).
		SetUniqueLimit(productDto.UniqueLimit).
		SetIsExpiring(productDto.IsExpiring)

	if productDto.LimitedTill != nil {
		query.SetLimitedTill(*productDto.LimitedTill)
	}

	if productDto.ExpiringTime != nil {
		query.SetExpiringTime(*productDto.ExpiringTime)
	}

	updatedProduct, err := query.Save(ctx)
	if err != nil {
		return nil, err
	}

	updatedProduct.Edges = productEnt.Edges

	if len(productDto.Bundles) > 0 {
		updatedProduct, err = r.updateBundles(ctx, tx, updatedProduct, productDto)
		if err != nil {
			return nil, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return updatedProduct, nil
}

func (r *productsRepo) updateBundles(
	ctx context.Context,
	tx *ent.Tx,
	productEnt *ent.Product,
	productDto *ProductDto,
) (
	*ent.Product, error,
) {
	bundlesMap := make(map[int64]float64)
	for _, bund := range productEnt.Edges.Bundles {
		bundlesMap[bund.ItemID] = bund.Amount
	}

	var updateBundles []BundleDto
	var createBundles []BundleDto
	var unchangedBundles []int64

	for _, bund := range productDto.Bundles {
		if _, ok := bundlesMap[bund.ItemID]; !ok {
			createBundles = append(createBundles, bund)
		} else if bund.Amount != bundlesMap[bund.ItemID] {
			updateBundles = append(updateBundles, bund)
		} else if bund.Amount == bundlesMap[bund.ItemID] {
			unchangedBundles = append(unchangedBundles, bund.ItemID)
		}
	}

	_, err := tx.Bundle.Delete().
		Where(
			bundle.ProductID(productEnt.ID),
			bundle.ItemIDNotIn(unchangedBundles...),
		).
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	err = tx.Bundle.MapCreateBulk(createBundles, func(create *ent.BundleCreate, i int) {
		create.SetAmount(createBundles[i].Amount).
			SetItemID(createBundles[i].ItemID).
			SetProductID(productEnt.ID)
	}).Exec(ctx)
	if err != nil {
		return nil, err
	}

	for _, updateBundle := range updateBundles {
		err = tx.Bundle.Update().Where(
			bundle.ProductID(productEnt.ID),
			bundle.ItemID(updateBundle.ItemID),
		).ClearDeletedAt().SetAmount(updateBundle.Amount).Exec(ctx)
		if err != nil {
			return nil, err
		}
	}

	var bundles []*ent.Bundle

	bundles, err = tx.Bundle.Query().Where(bundle.ProductID(productEnt.ID)).All(ctx)
	if err != nil {
		return nil, err
	}

	productEnt.Edges.Bundles = bundles

	return productEnt, nil
}

func (r *productsRepo) DeleteProduct(ctx context.Context, productID int64) error {
	return r.db.Product.DeleteOneID(productID).
		Exec(ctx)
}

func (r *productsRepo) GetProduct(ctx context.Context, productID int64) (*ent.Product, error) {
	return r.db.Product.Query().
		Where(product.ID(productID)).
		WithBundles().
		Only(ctx)
}

func (r *productsRepo) ListProducts(
	ctx context.Context, appID string, paginate *utils_v1.PaginateRequest,
) ([]*ent.Product, error) {
	return r.db.Product.Query().
		Where(
			product.AppID(appID),
			product.IDGT(paginate.GetFromId()),
		).Limit(int(paginate.GetLimit())).All(ctx)
}

func (r *productsRepo) CountProducts(ctx context.Context, appID string) (int32, error) {
	n, err := r.db.Product.Query().
		Where(product.AppID(appID)).
		Count(ctx)
	if err != nil {
		return 0, err
	}

	//nolint:gosec // pagination limit cannot hold more than int32
	return int32(n), nil
}
