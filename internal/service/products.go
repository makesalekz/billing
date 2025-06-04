package service

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	v1 "gitlab.calendaria.team/services/finance/billing/api/billing/v1"
	"gitlab.calendaria.team/services/finance/billing/internal/biz"
	"gitlab.calendaria.team/services/finance/billing/internal/data"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
	"gitlab.calendaria.team/services/utils/v2/auth"
)

type ProductService struct {
	v1.UnimplementedProductsServer
	uc *biz.ProductUseCase
}

func NewProductService(uc *biz.ProductUseCase) *ProductService {
	return &ProductService{uc: uc}
}

func (s *ProductService) CreateProduct(ctx context.Context, req *v1.CreateProductRequest) (*v1.ProductReply, error) {
	priceStr := req.GetProduct().GetPrice()
	priceDec, err := decimal.NewFromString(priceStr)
	if err != nil {
		return nil, v1.ErrorInvalidRequest("invalid price")
	}

	productDto := data.ProductDto{
		AppID:       req.GetProduct().GetAppId(),
		Name:        req.GetProduct().GetName(),
		Description: req.GetProduct().GetDescription(),
		Price:       priceDec,
		Currency:    req.GetProduct().GetCurrency(),
		IsActive:    req.GetProduct().GetIsActive(),
		IsLimited:   req.GetProduct().GetIsLimited(),
		Left:        req.GetProduct().GetLeft(),
		IsUnique:    req.GetProduct().GetIsUnique(),
		UniqueLimit: req.GetProduct().GetUniqueLimit(),
		IsExpiring:  req.GetProduct().GetIsExpiring(),
	}

	if req.GetProduct().LimitedTill != nil && req.GetProduct().GetLimitedTill() != "" {
		limitedTillStr := req.GetProduct().GetLimitedTill()

		var limitedTillTime time.Time

		limitedTillTime, err = time.Parse(limitedTillStr, time.RFC3339)
		if err != nil {
			return nil, v1.ErrorInvalidRequest("invalid limited till")
		}

		productDto.LimitedTill = &limitedTillTime
	}

	if req.GetProduct().ExpiringTime != nil && req.GetProduct().GetExpiringTime() != "" {
		expiringTimeStr := req.GetProduct().GetExpiringTime()

		var expiringTimeTime time.Time

		expiringTimeTime, err = time.Parse(expiringTimeStr, time.RFC3339)
		if err != nil {
			return nil, v1.ErrorInvalidRequest("invalid expiring time")
		}

		productDto.ExpiringTime = &expiringTimeTime
	}

	if len(req.GetProduct().GetBundles()) > 0 {
		for _, bundle := range req.GetProduct().GetBundles() {
			productDto.Bundles = append(
				productDto.Bundles, data.BundleDto{
					ItemID: bundle.GetItemId(),
					Amount: bundle.GetAmount(),
				},
			)
		}
	}

	product, err := s.uc.CreateProduct(ctx, productDto)
	if err != nil {
		return nil, err
	}

	return &v1.ProductReply{
		Product: biz.ReplyProduct(product),
	}, nil
}

func (s *ProductService) UpdateProduct(ctx context.Context, req *v1.UpdateProductRequest) (*v1.ProductReply, error) {
	priceStr := req.GetProduct().GetPrice()
	priceDec, err := decimal.NewFromString(priceStr)
	if err != nil {
		return nil, v1.ErrorInvalidRequest("invalid price")
	}

	productDto := data.ProductDto{
		Name:        req.GetProduct().GetName(),
		Description: req.GetProduct().GetDescription(),
		Price:       priceDec,
		Currency:    req.GetProduct().GetCurrency(),
		IsActive:    req.GetProduct().GetIsActive(),
		IsLimited:   req.GetProduct().GetIsLimited(),
		Left:        req.GetProduct().GetLeft(),
		IsUnique:    req.GetProduct().GetIsUnique(),
		UniqueLimit: req.GetProduct().GetUniqueLimit(),
		IsExpiring:  req.GetProduct().GetIsExpiring(),
	}

	if req.GetProduct().LimitedTill != nil && req.GetProduct().GetLimitedTill() != "" {
		limitedTillStr := req.GetProduct().GetLimitedTill()

		var limitedTillTime time.Time

		limitedTillTime, err = time.Parse(limitedTillStr, time.RFC3339)
		if err != nil {
			return nil, v1.ErrorInvalidRequest("invalid limited till")
		}

		productDto.LimitedTill = &limitedTillTime
	}

	if req.GetProduct().ExpiringTime != nil && req.GetProduct().GetExpiringTime() != "" {
		expiringTimeStr := req.GetProduct().GetExpiringTime()

		var expiringTimeTime time.Time

		expiringTimeTime, err = time.Parse(expiringTimeStr, time.RFC3339)
		if err != nil {
			return nil, v1.ErrorInvalidRequest("invalid expiring time")
		}

		productDto.ExpiringTime = &expiringTimeTime
	}

	if len(req.GetProduct().GetBundles()) > 0 {
		for _, bundle := range req.GetProduct().GetBundles() {
			productDto.Bundles = append(
				productDto.Bundles, data.BundleDto{
					ItemID: bundle.GetItemId(),
					Amount: bundle.GetAmount(),
				},
			)
		}
	}

	product, err := s.uc.UpdateProduct(ctx, req.GetId(), productDto)
	if err != nil {
		return nil, err
	}

	return &v1.ProductReply{
		Product: biz.ReplyProduct(product),
	}, nil
}

func (s *ProductService) DeleteProduct(ctx context.Context, req *v1.DeleteProductRequest) (
	*utils_v1.EmptyReply, error,
) {
	err := s.uc.DeleteProduct(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &utils_v1.EmptyReply{}, nil
}

func (s *ProductService) GetProduct(ctx context.Context, req *v1.GetProductRequest) (*v1.ProductReply, error) {
	product, err := s.uc.GetProduct(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &v1.ProductReply{Product: biz.ReplyProduct(product)}, nil
}

func (s *ProductService) ListProducts(ctx context.Context, req *v1.ListProductsRequest) (*v1.ListProductsReply, error) {
	appID := auth.GetAppIdFromContext(ctx)
	if appID == "" {
		return nil, v1.ErrorEmptyAppId("empty app id")
	}

	pagination := FormPaginateRequest(req.GetPagination())

	productList, err := s.uc.ListProducts(ctx, appID, pagination)
	if err != nil {
		return nil, err
	}

	return &v1.ListProductsReply{
		Products:   biz.ReplyProducts(productList.Products),
		Pagination: productList.PaginateReply,
	}, nil
}
