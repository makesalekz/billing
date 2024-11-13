package biz

import (
	"context"

	v1 "gitlab.calendaria.team/services/finance/invoices/api/invoices/v1"
	"gitlab.calendaria.team/services/finance/invoices/ent"
	"gitlab.calendaria.team/services/finance/invoices/internal/data"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
)

type ProductsList struct {
	Products      []*ent.Product
	PaginateReply *utils_v1.PaginateReply
}

type ProductUseCase struct {
	productRepo data.ProductRepo
}

func NewProductUseCase(productRepo data.ProductRepo) *ProductUseCase {
	return &ProductUseCase{
		productRepo: productRepo,
	}
}

func (uc *ProductUseCase) CreateProduct(ctx context.Context, productDto *data.ProductDto) (*ent.Product, error) {
	product, err := uc.productRepo.CreateProduct(ctx, productDto)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("failed to create product: %s", err.Error())
	}

	if productDto.OfferInAppleStore {

	}

	return product, nil
}

func (uc *ProductUseCase) UpdateProduct(
	ctx context.Context, productID int64, productDto *data.ProductDto,
) (*ent.Product, error) {
	productData, err := uc.productRepo.GetProduct(ctx, productID)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("failed to get product, err %s", err.Error())
	}

	product, err := uc.productRepo.UpdateProduct(ctx, productData, productDto)
	if err != nil {
		return nil, v1.ErrorInvalidRequest("failed to update product: %s", err.Error())
	}

	return product, nil
}

func (uc *ProductUseCase) DeleteProduct(ctx context.Context, productID int64) error {
	err := uc.productRepo.DeleteProduct(ctx, productID)
	if err != nil {
		return v1.ErrorInvalidRequest("failed to delete product: %s", err.Error())
	}

	return nil
}

func (uc *ProductUseCase) GetProduct(ctx context.Context, productID int64) (*ent.Product, error) {
	product, err := uc.productRepo.GetProduct(ctx, productID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, v1.ErrorNotFound("failed to get product: %s", err.Error())
		}

		return nil, v1.ErrorDatabaseQuery("failed to get product: %s", err.Error())
	}

	return product, nil
}

func (uc *ProductUseCase) ListProducts(
	ctx context.Context, appID string, paginate *utils_v1.PaginateRequest,
) (*ProductsList, error) {
	products, err := uc.productRepo.ListProducts(ctx, appID, paginate)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("failed to list products, err %s", err.Error())
	}

	total, err := uc.productRepo.CountProducts(ctx, appID)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("failed to count products, err %s", err.Error())
	}

	paginateReply := &utils_v1.PaginateReply{Total: &total}
	if len(products) > 0 {
		paginateReply.FromId = &products[len(products)-1].ID
	}

	return &ProductsList{
		Products:      products,
		PaginateReply: paginateReply,
	}, nil
}
